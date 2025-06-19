package com.vkontech.orderservice.kafka;

import com.vkontech.orderservice.service.OrderService;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class InventoryReservedConsumer {

    private static final Logger logger = LoggerFactory.getLogger(InventoryReservedConsumer.class);

    private final OrderService orderService;
    private final ObjectMapper objectMapper = new ObjectMapper();

    public InventoryReservedConsumer(OrderService orderService) {
        this.orderService = orderService;
    }

    @KafkaListener(topics = "InventoryReserved", groupId = "order-service-inventory-consumer")
    public void consume(ConsumerRecord<String, String> record) {
        try {
            logger.info("=== DEMO: Received InventoryReserved event from partition={} offset={} ===", 
                record.partition(), record.offset());

            JsonNode json = objectMapper.readTree(record.value());
            UUID orderId = UUID.fromString(json.get("order_id").asText());

            logger.info("=== DEMO: Processing inventory reservation for order {} ===", orderId);

            orderService.markOrderAsInventoryReserved(orderId);
            
            logger.info("=== DEMO: Order {} successfully updated to INVENTORY_RESERVED ===", orderId);
        } catch (Exception e) {
            logger.error("=== DEMO: Failed to process InventoryReserved message ===", e);
        }
    }
}
