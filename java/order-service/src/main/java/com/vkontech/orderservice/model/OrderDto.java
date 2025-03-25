package com.vkontech.orderservice.model;

public class OrderDto {
    private String customerId;
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
