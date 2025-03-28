package com.vkontech.orderservice.controller;

import com.vkontech.orderservice.model.CreateOrderRequest;
import com.vkontech.orderservice.model.CreateOrderResponse;
import com.vkontech.orderservice.service.OrderService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@RestController
@RequestMapping("/orders")
public class OrderController {

    private final OrderService orderService;

    public OrderController(OrderService orderService) {
        this.orderService = orderService;
    }

    @PostMapping
    public ResponseEntity<?> createOrder(@RequestBody CreateOrderRequest createOrderRequest) {
        try {
            CreateOrderResponse result = orderService.createOrder(createOrderRequest);
            return ResponseEntity.status(201).body(result);
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }
}
