import logging
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Directly set OTLP exporter configuration
OTLP_ENDPOINT = "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp"
OTLP_HEADERS = {"Authorization": "Basic glc_eyJvIjoiMTM3MzU3OCIsIm4iOiJzdGFjay0xMTk3MTY3LW90bHAtd3JpdGUtbXktb3RscC1hY2Nlc3MtdG9rZW4iLCJrIjoicXZYaDNGOGg1cVY2eTJSM29wNzY4NmdNIiwibSI6eyJyIjoicHJvZC1ldS13ZXN0LTIifX0="}

# Log the OTLP configuration
logger.info(f"OTLP Endpoint: {OTLP_ENDPOINT}")
logger.info(f"OTLP Headers: {OTLP_HEADERS['Authorization'][:20]}... (truncated for security)")

# Set up OpenTelemetry tracing
trace.set_tracer_provider(TracerProvider())
tracer = trace.get_tracer_provider().get_tracer(__name__)

# Configure OTLP exporter
otlp_exporter = OTLPSpanExporter(endpoint=OTLP_ENDPOINT, headers=OTLP_HEADERS)
trace.get_tracer_provider().add_span_processor(BatchSpanProcessor(otlp_exporter))

def main():
    # Create and send a simple span
    with tracer.start_as_current_span("test_span") as span:
        span.set_attribute("test.key", "test_value")
        logger.info("Span created and sent!")

    logger.info("Done!")

if __name__ == "__main__":
    main()
