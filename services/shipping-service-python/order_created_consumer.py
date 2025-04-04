import os
import json
from kafka import KafkaConsumer, KafkaProducer
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import SpanKind
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

# Read OTLP Headers from Environment
OTLP_HEADERS = os.getenv("OTEL_EXPORTER_OTLP_HEADERS")

# Initialize tracing with correct service name
SERVICE_NAME = "shipping_service"
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({"service.name": SERVICE_NAME}))
)
tracer = trace.get_tracer(__name__)

# Configure OTLP HTTP Exporter for Grafana Cloud
otlp_exporter = OTLPSpanExporter(
    endpoint="https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces",
    headers=dict([OTLP_HEADERS.split("=", 1)]) if OTLP_HEADERS else {},
)

# Attach Exporter to Provider
trace.get_tracer_provider().add_span_processor(BatchSpanProcessor(otlp_exporter))

# Kafka Propagation Utilities
propagator = TraceContextTextMapPropagator()

def get_kafka_producer():
    """Returns a Kafka producer"""
    return KafkaProducer(
        bootstrap_servers='localhost:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )

def get_kafka_consumer():
    """Returns a Kafka consumer"""
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

def extract_trace_context(kafka_headers):
    """Extracts OpenTelemetry trace context from Kafka headers"""
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
                "order.id": order_id
            }
        ) as span:
            print(f"Processing order {order_id}...")

            packaging_event = {
                "order_id": order_id,
                "status": "PACKAGED"
            }

            producer.send('PackagingCompleted', value=packaging_event)
            producer.flush()

            print(f"Published PackagingCompleted event for order {order_id}")
