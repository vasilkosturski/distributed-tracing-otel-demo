package com.vkontech.orderservice.service;

import com.vkontech.orderservice.model.OrderDto;
import com.vkontech.orderservice.model.OrderResult;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.Tracer;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaTemplate;
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

    public OrderResult createOrder(OrderDto orderDto) {
        UUID orderId = UUID.randomUUID();
        String status = "CREATED";

        Span span = tracer.spanBuilder("create_order").startSpan();
        try {
            // Insert into DB
            jdbcTemplate.update(
                    "INSERT INTO orders (id, customer_id, product_id, quantity, status) VALUES (?, ?, ?, ?, ?)",
                    orderId,
                    orderDto.getCustomerId(),
                    orderDto.getProductId(),
                    orderDto.getQuantity(),
                    status
            );

            // Prepare Kafka event
            Map<String, Object> event = new HashMap<>();
            event.put("order_id", orderId);
            event.put("customer_id", orderDto.getCustomerId());
            event.put("product_id", orderDto.getProductId());
            event.put("quantity", orderDto.getQuantity());
            event.put("status", status);

            kafkaTemplate.send("OrderCreated", event);

            return new OrderResult(orderId.toString(), status);
        } catch (Exception e) {
            span.recordException(e);
            throw new RuntimeException("Failed to create order: " + e.getMessage(), e);
        } finally {
            span.end();
        }
    }
}
