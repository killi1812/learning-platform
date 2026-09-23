package no.DAT250.LearningPlatform.dto;

public record AuthResponse(
    boolean authenticated,
    String username,
    String message
) {}
