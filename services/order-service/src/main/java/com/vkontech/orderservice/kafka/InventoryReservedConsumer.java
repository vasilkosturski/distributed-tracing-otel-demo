package com.vkontech.orderservice.kafka;

import com.vkontech.orderservice.service.OrderService;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class InventoryReservedConsumer {

    private final OrderService orderService;
    private final ObjectMapper objectMapper = new ObjectMapper();

    public InventoryReservedConsumer(OrderService orderService) {
        this.orderService = orderService;
    }

    @KafkaListener(topics = "InventoryReserved", groupId = "order-service-inventory-consumer")
    public void consume(ConsumerRecord<String, String> record) {
        try {
            JsonNode json = objectMapper.readTree(record.value());
            UUID orderId = UUID.fromString(json.get("order_id").asText());

            orderService.markOrderAsInventoryReserved(orderId);
            System.out.println("Order " + orderId + " updated to INVENTORY_RESERVED");
        } catch (Exception e) {
            System.err.println("Failed to process InventoryReserved message: " + e.getMessage());
        }
    }
}
