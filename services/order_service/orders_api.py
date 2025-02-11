from flask import Flask, request, jsonify
from kafka import KafkaProducer
import psycopg2
import json
import uuid
from opentelemetry import trace
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

app = Flask(__name__)

trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({SERVICE_NAME: "order_service"}))
)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
trace.get_tracer_provider().add_span_processor(
    BatchSpanProcessor(otlp_exporter)
)
tracer = trace.get_tracer(__name__)

def get_db_connection():
    return psycopg2.connect(
        dbname='orders_db', user='postgres', password='password', host='localhost'
    )

def get_kafka_producer():
    return KafkaProducer(
        bootstrap_servers='localhost:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )

@app.route('/orders', methods=['POST'])
def handle_order():
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
