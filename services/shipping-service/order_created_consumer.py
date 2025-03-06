from kafka import KafkaConsumer
import json
import time
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

# Initialize tracing
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "shipping_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)
propagator = TraceContextTextMapPropagator()

def extract_trace_context(kafka_headers):
    """
    Extracts the tracing context from Kafka headers.
    """
    if kafka_headers:
        carrier = {key: value.decode("utf-8") for key, value in kafka_headers if isinstance(value, bytes)}
        return propagator.extract(carrier)
    return trace.INVALID_SPAN_CONTEXT  # Fallback to an invalid span context if no headers exist

def get_kafka_consumer():
    """
    Returns a configured Kafka consumer.
    """
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

if __name__ == '__main__':
    KafkaInstrumentor().instrument()

    consumer = get_kafka_consumer()
    print("Shipping Service is running and listening for OrderCreated events...")

    for message in consumer:
        context = extract_trace_context(message.headers)

        with tracer.start_as_current_span("process_order", context=context):
            order_event = message.value
            order_id = order_event.get('order_id')

            print(f"Processing order {order_id}...")
            time.sleep(2)
