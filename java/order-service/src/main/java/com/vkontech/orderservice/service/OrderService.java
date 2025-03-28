package com.vkontech.orderservice.service;

import com.vkontech.orderservice.model.CreateOrderRequest;
import com.vkontech.orderservice.model.CreateOrderResponse;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.Tracer;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.support.SendResult;
import org.springframework.stereotype.Service;

import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

@Service
public class OrderService {

    private final JdbcTemplate jdbcTemplate;
    private final KafkaTemplate<String, Object> kafkaTemplate;
    private final Tracer tracer;

    public OrderService(JdbcTemplate jdbcTemplate,
                        KafkaTemplate<String, Object> kafkaTemplate,
                        Tracer tracer) {
        this.jdbcTemplate = jdbcTemplate;
        this.kafkaTemplate = kafkaTemplate;
        this.tracer = tracer;
    }

    public CreateOrderResponse createOrder(CreateOrderRequest createOrderRequest) {
        UUID orderId = UUID.randomUUID();
        String status = "CREATED";

        Span span = tracer.spanBuilder("create_order").startSpan();
        try {
            // Insert into DB
            jdbcTemplate.update(
                    "INSERT INTO orders (id, customer_id, product_id, quantity, status) VALUES (?, ?, ?, ?, ?)",
                    orderId,
                    createOrderRequest.getCustomerId(),
                    createOrderRequest.getProductId(),
                    createOrderRequest.getQuantity(),
                    status
            );

            // Prepare Kafka event
            Map<String, Object> event = new HashMap<>();
            event.put("order_id", orderId);
            event.put("customer_id", createOrderRequest.getCustomerId());
            event.put("product_id", createOrderRequest.getProductId());
            event.put("quantity", createOrderRequest.getQuantity());
            event.put("status", status);

            SendResult<String, Object> result = kafkaTemplate.send("OrderCreated", event).get();

            System.out.println("Message sent successfully to topic: " + result.getRecordMetadata().topic());
            System.out.println("Partition: " + result.getRecordMetadata().partition());
            System.out.println("Offset: " + result.getRecordMetadata().offset());

            return new CreateOrderResponse(orderId.toString(), status);
        } catch (Exception e) {
            span.recordException(e);
            throw new RuntimeException("Failed to create order: " + e.getMessage(), e);
        } finally {
            span.end();
        }
    }
}
