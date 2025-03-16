from kafka import KafkaConsumer
import psycopg2
import json
from opentelemetry import trace
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import SpanKind
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

# Initialize tracing
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "order_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="localhost:4317", insecure=True)
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)
propagator = TraceContextTextMapPropagator()

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

# Extract trace context from Kafka headers
def extract_trace_context(kafka_headers):
    carrier = {key: value.decode("utf-8") for key, value in kafka_headers if isinstance(value, bytes)}
    return propagator.extract(carrier)

# Processing PackagingCompleted events
def process_packaging_completed():
    consumer = get_kafka_consumer()
    print("Order Consumer Service is listening for PackagingCompleted events...")

    for message in consumer:
        context = extract_trace_context(message.headers)
        event = message.value
        order_id = event.get('order_id')

        with tracer.start_as_current_span(
            "PackagingCompleted process",
            context=context,
            kind=SpanKind.CONSUMER,
            attributes={
                "messaging.system": "kafka",
                "messaging.destination.name": "PackagingCompleted",
                "messaging.kafka.partition": message.partition,
                "messaging.message.id": message.offset,
                "order.id": order_id
            }
        ) as span:
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
