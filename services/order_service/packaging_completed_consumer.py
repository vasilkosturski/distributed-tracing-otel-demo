from kafka import KafkaConsumer
import psycopg2
import json
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# Tracing setup
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "order_consumer_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)

# Kafka Consumer
def get_kafka_consumer():
    return KafkaConsumer(
        'PackagingCompleted',
        bootstrap_servers='localhost:9092',
        value_deserializer=lambda m: json.loads(m.decode('utf-8')),
        group_id='order-consumer-group'
    )

# Database connection
def get_db_connection():
    return psycopg2.connect(
        dbname='orders_db', user='postgres', password='password', host='localhost'
    )

# Processing PackagingCompleted events
def process_packaging_completed():
    consumer = get_kafka_consumer()
    print("Order Consumer Service is listening for PackagingCompleted events...")

    for message in consumer:
        event = message.value
        order_id = event.get('order_id')

        with tracer.start_as_current_span("update_order_status", attributes={"order_id": order_id}):
            conn = get_db_connection()
            cursor = conn.cursor()
            try:
                cursor.execute(
                    "UPDATE orders SET status = 'PACKAGED' WHERE id = %s",
                    (order_id,)
                )
                conn.commit()
                print(f"Order {order_id} status updated to PACKAGED")
            except Exception as e:
                conn.rollback()
                print(f"Failed to update order {order_id}: {e}")
            finally:
                cursor.close()
                conn.close()

if __name__ == '__main__':
    KafkaInstrumentor().instrument()
    Psycopg2Instrumentor().instrument()
    process_packaging_completed()