package com.vkontech.orderservice.kafka;

import com.vkontech.orderservice.service.OrderService;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class PackagingCompletedConsumer {

    private final OrderService orderService;
    private final ObjectMapper objectMapper = new ObjectMapper();

    public PackagingCompletedConsumer(OrderService orderService) {
        this.orderService = orderService;
    }

    @KafkaListener(topics = "PackagingCompleted", groupId = "order-service-packaging-consumer")
    public void consume(ConsumerRecord<String, String> record) {
        try {
            JsonNode json = objectMapper.readTree(record.value());
            UUID orderId = UUID.fromString(json.get("order_id").asText());

            orderService.markOrderAsPackaged(orderId);
            System.out.println("Order " + orderId + " updated to PACKAGED");
        } catch (Exception e) {
            System.err.println("Failed to process PackagingCompleted message: " + e.getMessage());
        }
    }
}
