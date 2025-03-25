package com.vkontech.orderservice.model;

public class OrderResult {
    private String orderId;
    private String status;

    public OrderResult(String orderId, String status) {
        this.orderId = orderId;
        this.status = status;
    }

    public String getOrderId() {
        return orderId;
    }

    public String getStatus() {
        return status;
    }
}
