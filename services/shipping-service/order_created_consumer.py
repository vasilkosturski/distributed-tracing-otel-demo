from kafka import KafkaConsumer
import json
import time
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "shipping_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)

def consume_hook(span, record, args, kwargs):
    """
    Hook that runs when KafkaInstrumentation creates the 'receive' span.
    We use it to ensure the 'process' span is a child of 'receive'.
    """
    if span and span.is_recording():
        order_event = record.value  # ✅ FIXED: record.value is already a dictionary
        order_id = order_event.get('order_id')

        # ✅ Run "process" inside "receive" so it becomes its child
        with trace.use_span(span, end_on_exit=False):
            with tracer.start_as_current_span("OrderCreated process"):
                print(f"Processing order {order_id}...")
                time.sleep(2)  # Simulate processing delay

def get_kafka_consumer():
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

if __name__ == '__main__':
    # ✅ Instrument Kafka with the consume hook
    KafkaInstrumentor().instrument(consume_hook=consume_hook)

    consumer = get_kafka_consumer()
    print("Shipping Service is running and listening for OrderCreated events...")

    for message in consumer:
        # ✅ We do not need to manually extract context or create spans here
        pass  # The consume_hook handles processing inside the correct span