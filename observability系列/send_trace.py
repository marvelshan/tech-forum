from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# 設定 OpenTelemetry
trace.set_tracer_provider(TracerProvider())
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4317/", insecure=True)
span_processor = BatchSpanProcessor(otlp_exporter)
trace.get_tracer_provider().add_span_processor(span_processor)

# 產生一個 trace
tracer = trace.get_tracer(__name__)
with tracer.start_as_current_span("test-span"):
    span.set_attribute("user.id", "12345")
    span.set_attribute("http.method", "GET")
    print("Hello from OpenTelemetry!")