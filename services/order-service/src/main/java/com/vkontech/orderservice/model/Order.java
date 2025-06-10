package com.vkontech.orderservice.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.UUID;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class Order {
    private UUID id;
    
    @JsonProperty("customer_id")
    private UUID customerId;
    
    @JsonProperty("product_id")
    private UUID productId;
    
    private int quantity;
    private String status;
} 