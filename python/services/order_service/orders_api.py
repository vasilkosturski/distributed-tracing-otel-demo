import os
import json
import uuid
import psycopg2
from flask import Flask, request, jsonify
from kafka import KafkaProducer
from opentelemetry import trace
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# Flask App Initialization
app = Flask(__name__)

# Read OTLP Headers from Environment
OTLP_HEADERS = os.getenv("OTEL_EXPORTER_OTLP_HEADERS")

# Set up OpenTelemetry Tracer with service name
SERVICE_NAME = "order_service"
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

def get_db_connection():
    """Returns a PostgreSQL database connection"""
    return psycopg2.connect(
        dbname='orders_db', user='postgres', password='password', host='localhost'
    )

def get_kafka_producer():
    """Returns a Kafka producer"""
    return KafkaProducer(
        bootstrap_servers='localhost:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )

@app.route('/orders', methods=['POST'])
def handle_order():
    """Handles order creation, inserts into DB, and sends an event to Kafka"""
    producer = get_kafka_producer()
    data = request.json
    order_id = str(uuid.uuid4())
    customer_id = data.get('customer_id')
    product_id = data.get('product_id')
    quantity = data.get('quantity')
    status = 'CREATED'

    with tracer.start_as_current_span("create_order"):
        conn = get_db_connection()
        cursor = conn.cursor()
        try:
            cursor.execute(
                "INSERT INTO orders (id, customer_id, product_id, quantity, status) VALUES (%s, %s, %s, %s, %s)",
                (order_id, customer_id, product_id, quantity, status)
            )
            conn.commit()

            event = {
                'order_id': order_id,
                'customer_id': customer_id,
                'product_id': product_id,
                'quantity': quantity,
                'status': status
            }
            producer.send('OrderCreated', event)
            producer.flush()

            return jsonify({'order_id': order_id, 'status': status}), 201
        except Exception as e:
            conn.rollback()
            return jsonify({'error': str(e)}), 500
        finally:
            cursor.close()
            conn.close()

if __name__ == '__main__':
    FlaskInstrumentor().instrument_app(app)
    Psycopg2Instrumentor().instrument()
    KafkaInstrumentor().instrument()
    app.run(host='0.0.0.0', port=5000)
