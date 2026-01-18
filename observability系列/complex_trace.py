from opentelemetry import trace
from opentelemetry.trace import StatusCode
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
import time
import random

# --- 初始化設定 ---
provider = TracerProvider()
trace.set_tracer_provider(provider)
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4317/", insecure=True)
provider.add_span_processor(BatchSpanProcessor(otlp_exporter))
tracer = trace.get_tracer(__name__)
def process_order(order_id):
    # 父跨度：整個訂單工作流
    with tracer.start_as_current_span("Execute-Order-Workflow", attributes={"order.id": order_id}) as root:
        root.add_event("workflow_started")

        # 1. 欺詐檢測 (Fraud Detection)
        with tracer.start_as_current_span("Fraud-Detection") as span:
            span.set_attribute("user.ip", "192.168.1.50")
            time.sleep(0.03)
            span.add_event("risk_score_calculated", {"score": 15})
            span.set_status(StatusCode.OK)

        # 2. 庫存查詢 (Inventory Check)
        with tracer.start_as_current_span("Inventory-Check") as span:
            time.sleep(0.08)
            span.add_event("inventory_reserved", {"sku": "GTX-4090", "count": 1})
            span.set_status(StatusCode.OK)

        # 3. 支付處理 (Payment Gateway)
        with tracer.start_as_current_span("Payment-Gateway") as span:
            time.sleep(0.15)
            if random.random() < 0.1: # 模擬 10% 支付延遲
                span.add_event("gateway_latency_detected")
                time.sleep(0.2)
            span.add_event("payment_captured")
            span.set_status(StatusCode.OK)

        # 4. 會員點數更新 (Loyalty-Points)
        with tracer.start_as_current_span("Loyalty-Points-Update") as span:
            time.sleep(0.04)
            span.add_event("points_added", {"points": 100})
            span.set_status(StatusCode.OK)

        # 5. 物流預約 (Shipping-Service)
        with tracer.start_as_current_span("Shipping-Service") as span:
            time.sleep(0.1)
            # 隨機模擬物流 API 失敗
            if random.random() < 0.2:
                err_msg = "Carrier API Timeout"
                span.record_exception(RuntimeError(err_msg))
                span.set_status(StatusCode.ERROR, err_msg)
            else:
                span.add_event("shipping_label_created")
                span.set_status(StatusCode.OK)

        # 6. 電子發票生成 (Invoice-Generation)
        with tracer.start_as_current_span("Invoice-Generation") as span:
            time.sleep(0.07)
            span.add_event("pdf_rendered")
            span.add_event("email_queued")
            span.set_status(StatusCode.OK)

        # 7. 顧客通知 (Notification-Service)
        with tracer.start_as_current_span("Notification-Service") as span:
            # 模擬異步通知
            channels = ["email", "sms", "push"]
            for channel in channels:
                span.add_event(f"sending_{channel}")
                time.sleep(0.02)
            span.set_status(StatusCode.OK)

        root.add_event("workflow_finished")

if __name__ == "__main__":
    total_runs = 2  # 依照你的需求改為 20
    print(f"🚀 正在開始產生 {total_runs} 筆 Trace 資料...")

    for i in range(1, total_runs + 1):
        # 產生一個模擬的訂單 ID，例如 ORD-1001
        dynamic_order_id = f"ORD-{1000 + i}"
        
        print(f"[{i}/{total_runs}] 正在執行訂單流程: {dynamic_order_id}...")
        
        # 修正重點：傳入動態產生的 order_id
        process_order(dynamic_order_id)
        
        # 稍微停頓一下，讓 Grafana 時間軸分開
        time.sleep(0.2)

    # 確保資料完整送出
    provider.shutdown()
    print("\n✅ 所有資料已發送完成！請至 Grafana 查看。")