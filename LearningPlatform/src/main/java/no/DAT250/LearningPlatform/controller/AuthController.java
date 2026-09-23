package no.DAT250.LearningPlatform.controller;

import jakarta.validation.Valid;
import no.DAT250.LearningPlatform.dto.AuthResponse;
import no.DAT250.LearningPlatform.dto.LoginRequest;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@RestController
@RequestMapping("/api/auth")
public class AuthController {

    @GetMapping("/status")
    public ResponseEntity<Map<String, Object>> status() {
        return ResponseEntity.ok(Map.of(
            "status", "UP",
            "service", "LearningPlatform Backend",
            "timestamp", System.currentTimeMillis()
        ));
    }

    @PostMapping("/login")
    public ResponseEntity<AuthResponse> login(@Valid @RequestBody LoginRequest request) {
        if ("admin".equals(request.username()) && "password".equals(request.password())) {
            return ResponseEntity.ok(new AuthResponse(
                true,
                request.username(),
                "Authentication successful! Backend connection verified."
            ));
        }

        if ("user".equals(request.username()) && "password".equals(request.password())) {
            return ResponseEntity.ok(new AuthResponse(
                true,
                request.username(),
                "Authentication successful! Backend connection verified."
            ));
        }

        return ResponseEntity.status(HttpStatus.UNAUTHORIZED).body(new AuthResponse(
            false,
            request.username(),
            "Invalid credentials. Try admin / password"
        ));
    }
}
