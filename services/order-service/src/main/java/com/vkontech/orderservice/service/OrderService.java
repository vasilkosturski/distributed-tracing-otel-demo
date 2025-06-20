package com.vkontech.orderservice.service;

import com.vkontech.orderservice.kafka.OrderCreatedEvent;
import com.vkontech.orderservice.model.CreateOrderRequest;
import com.vkontech.orderservice.model.CreateOrderResponse;
import com.vkontech.orderservice.model.Order;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Scope;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.support.SendResult;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.UUID;

@Service
public class OrderService {

    private static final Logger logger = LoggerFactory.getLogger(OrderService.class);

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
        try (Scope ignored = span.makeCurrent()) {
            span.setAttribute("order.id", orderId.toString());
            span.setAttribute("customer.id", createOrderRequest.getCustomerId().toString());
            span.setAttribute("product.id", createOrderRequest.getProductId().toString());
            span.setAttribute("order.quantity", (long) createOrderRequest.getQuantity());
            span.setAttribute("order.status", status);

            logger.info("=== DEMO: Starting order creation for customer {} ===", 
                createOrderRequest.getCustomerId());
            logger.info("=== DEMO: Generated order ID: {} ===", orderId);

            jdbcTemplate.update(
                    "INSERT INTO orders (id, customer_id, product_id, quantity, status) VALUES (?, ?, ?, ?, ?)",
                    orderId,
                    createOrderRequest.getCustomerId(),
                    createOrderRequest.getProductId(),
                    createOrderRequest.getQuantity(),
                    status
            );

            logger.info("=== DEMO: Order {} saved to database ===", orderId);

            OrderCreatedEvent event = new OrderCreatedEvent(
                    orderId,
                    createOrderRequest.getCustomerId().toString(),
                    createOrderRequest.getProductId().toString(),
                    createOrderRequest.getQuantity()
            );

            logger.info("=== DEMO: Publishing OrderCreated event to Kafka ===");

            SendResult<String, Object> result = kafkaTemplate.send("OrderCreated", event).get();

            logger.info("=== DEMO: Event published to topic={} partition={} offset={} ===",
                    result.getRecordMetadata().topic(),
                    result.getRecordMetadata().partition(),
                    result.getRecordMetadata().offset());

            logger.info("=== DEMO: Order creation completed successfully ===");
            return new CreateOrderResponse(orderId.toString(), status);
            
        } catch (Exception e) {
            logger.error("=== DEMO: Failed to create order ===", e);
            span.setStatus(StatusCode.ERROR, "Order creation failed");
            span.recordException(e);
            throw new RuntimeException("Failed to create order: " + e.getMessage(), e);
        } finally {
            span.end();
        }
    }

    public void markOrderAsInventoryReserved(UUID orderId) {
        logger.info("=== DEMO: Marking order {} as INVENTORY_RESERVED ===", orderId);
        jdbcTemplate.update(
                "UPDATE orders SET status = ? WHERE id = ?",
                "INVENTORY_RESERVED", orderId
        );
        logger.info("=== DEMO: Order {} status updated successfully ===", orderId);
    }

    public List<Order> getAllOrders() {
        logger.info("=== DEMO: Fetching all orders from database ===");
        List<Order> orders = jdbcTemplate.query(
                "SELECT id, customer_id, product_id, quantity, status FROM orders ORDER BY id",
                (rs, rowNum) -> new Order(
                        UUID.fromString(rs.getString("id")),
                        UUID.fromString(rs.getString("customer_id")),
                        UUID.fromString(rs.getString("product_id")),
                        rs.getInt("quantity"),
                        rs.getString("status")
                )
        );
        logger.info("=== DEMO: Retrieved {} orders ===", orders.size());
        return orders;
    }
}
