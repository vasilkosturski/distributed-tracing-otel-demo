from kafka import KafkaConsumer, KafkaProducer
import json
import time
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import SpanKind
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

# Initialize tracing
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "shipping_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="localhost:4317", insecure=True)
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)
propagator = TraceContextTextMapPropagator()

def get_kafka_producer():
    return KafkaProducer(
        bootstrap_servers='localhost:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )

def get_kafka_consumer():
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

def extract_trace_context(kafka_headers):
    carrier = {key: value.decode("utf-8") for key, value in kafka_headers if isinstance(value, bytes)}
    return propagator.extract(carrier)

if __name__ == '__main__':
    KafkaInstrumentor().instrument()

    consumer = get_kafka_consumer()
    producer = get_kafka_producer()

    print("Shipping Service is listening for OrderCreated events...")

    for message in consumer:
        context = extract_trace_context(message.headers)

        order_event = message.value
        order_id = order_event.get('order_id')

        with tracer.start_as_current_span(
                "OrderCreated process",
                context=context,
                kind=SpanKind.CONSUMER,
                attributes={
                    "messaging.system": "kafka",
                    "messaging.destination.name": "OrderCreated",
                    "messaging.kafka.partition": message.partition,
                    "messaging.message.id": message.offset,
                    "order.id": order_event.get('order_id')
                }
        ) as span:
            print(f"Processing order {order_id}...")

            time.sleep(1)

            packaging_event = {
                "order_id": order_id,
                "status": "PACKAGED"
            }

            producer.send('PackagingCompleted', value=packaging_event)
            producer.flush()

            print(f"Published PackagingCompleted event for order {order_id}")
