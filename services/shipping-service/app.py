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

        with tracer.start_as_current_span("process_order_shipping", attributes={"order_id": order_id}):
            print(f"Processing order {order_id}...")
            time.sleep(2)  # Simulate packaging delay

            packaging_completed_event = {
                'order_id': order_id,
                'status': 'PACKAGED'
            }
            producer.send('PackagingCompleted', packaging_completed_event)
            producer.flush()

            print(f"Packaging completed for order {order_id}")

if __name__ == '__main__':
    KafkaInstrumentor().instrument()
    process_orders()
