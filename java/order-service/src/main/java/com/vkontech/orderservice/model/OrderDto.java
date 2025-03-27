package com.vkontech.orderservice.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class OrderDto {
    @JsonProperty("customer_id")
    private String customerId;

    @JsonProperty("product_id")
    private String productId;

    private int quantity;

    // Getters and setters (or use Lombok if preferred)
    public String getCustomerId() { return customerId; }
    public void setCustomerId(String customerId) { this.customerId = customerId; }

    public String getProductId() { return productId; }
    public void setProductId(String productId) { this.productId = productId; }

    public int getQuantity() { return quantity; }
    public void setQuantity(int quantity) { this.quantity = quantity; }
}
