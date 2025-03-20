import logging
import os
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define service name
SERVICE_NAME = "minimal_otel_service"

# Read OTLP headers from environment variable
OTLP_HEADERS = os.getenv("OTEL_EXPORTER_OTLP_HEADERS")

# Set up OpenTelemetry Tracer with service name
trace.set_tracer_provider(
    TracerProvider(resource=Resource.create({"service.name": SERVICE_NAME}))
)
tracer = trace.get_tracer(__name__)

# Configure OTLP HTTP Exporter to send data to Grafana Cloudpyt
otlp_exporter = OTLPSpanExporter(
    endpoint="https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces",
    headers=dict([OTLP_HEADERS.split("=", 1)]) if OTLP_HEADERS else {},
)
trace.get_tracer_provider().add_span_processor(BatchSpanProcessor(otlp_exporter))

def main():
    logger.info(f"Sending a test span from service: {SERVICE_NAME} to Jaeger...")

    # Create and send a simple span
    with tracer.start_as_current_span("test_span") as span:
        span.set_attribute("example.key", "example_value")
        logger.info("Span created and sent!")

    logger.info("Done!")

if __name__ == "__main__":
    main()
