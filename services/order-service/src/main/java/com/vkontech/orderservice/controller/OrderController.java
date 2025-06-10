package com.vkontech.orderservice.controller;

import com.vkontech.orderservice.model.CreateOrderRequest;
import com.vkontech.orderservice.model.CreateOrderResponse;
import com.vkontech.orderservice.model.Order;
import com.vkontech.orderservice.service.OrderService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/orders")
public class OrderController {

    private final OrderService orderService;

    public OrderController(OrderService orderService) {
        this.orderService = orderService;
    }

    @GetMapping
    public ResponseEntity<List<Order>> getAllOrders() {
        try {
            List<Order> orders = orderService.getAllOrders();
            return ResponseEntity.ok(orders);
        } catch (Exception e) {
            return ResponseEntity.internalServerError().build();
        }
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
