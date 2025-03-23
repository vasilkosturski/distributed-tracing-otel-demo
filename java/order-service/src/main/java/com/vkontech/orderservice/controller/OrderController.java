import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/orders")
public class OrderController {

    @PostMapping
    public ResponseEntity<String> createOrder(@RequestBody OrderDto order) {
        // TODO: publish to Kafka
        return ResponseEntity.ok("Order received");
    }
}
