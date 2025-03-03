from kafka import KafkaConsumer, KafkaProducer
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

def get_kafka_consumer():
    return KafkaConsumer(
        'OrderCreated',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='shipping-service-group'
    )

def process_orders():
    consumer = get_kafka_consumer()
    print("Shipping Service is running and listening for OrderCreated events...")

    for message in consumer:
        order_event = message.value
        order_id = order_event.get('order_id')

        print(f"Received order {order_id}...")  # ✅ Auto-traced by KafkaInstrumentation
        time.sleep(2)  # Simulate processing delay

if __name__ == '__main__':
    KafkaInstrumentor().instrument()
    process_orders()
