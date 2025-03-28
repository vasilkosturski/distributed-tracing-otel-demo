package com.vkontech.orderservice.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;

@Data
public class CreateOrderRequest {
    @JsonProperty("customer_id")
    private String customerId;

    @JsonProperty("product_id")
    private String productId;

    private int quantity;
}
