from kafka import KafkaConsumer, KafkaProducer
import json
import time
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "shipping_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)

def get_kafka_consumer():
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

def get_kafka_producer():
    return KafkaProducer(
        bootstrap_servers='localhost:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )

def process_orders():
    consumer = get_kafka_consumer()
    producer = get_kafka_producer()

    print("Shipping Service is running and listening for OrderCreated events...")

    for message in consumer:
        order_event = message.value
        order_id = order_event.get('order_id')

        # ✅ Extract trace context from Kafka headers
        headers = {key: value.decode('utf-8') for key, value in message.headers}
        ctx = TraceContextTextMapPropagator().extract(headers)

        # ✅ Start a new span for "OrderCreated receive" using the extracted context
        with tracer.start_as_current_span("OrderCreated receive", context=ctx) as consumer_span:
            current_span = trace.get_current_span()
            print(f"🛠 DEBUG: Active Span (should be OrderCreated receive): {current_span.get_span_context()}")

            # ✅ Now "process_order_shipping" should correctly be a child of "OrderCreated receive"
            with tracer.start_as_current_span(
                "process_order_shipping",
                context=trace.set_span_in_context(consumer_span),
                attributes={"order_id": order_id}
            ):
                print(f"Processing order {order_id}...")
                time.sleep(2)  # Simulate packaging delay

                packaging_completed_event = {'order_id': order_id}

                # ✅ Inject trace context into Kafka headers for next service
                new_headers = {}
                TraceContextTextMapPropagator().inject(new_headers)
                kafka_headers = [(k, v.encode('utf-8')) for k, v in new_headers.items()]

                producer.send('PackagingCompleted', value=packaging_completed_event, headers=kafka_headers)
                producer.flush()

                print(f"Packaging completed for order {order_id}")

if __name__ == '__main__':
    KafkaInstrumentor().instrument()
    process_orders()
