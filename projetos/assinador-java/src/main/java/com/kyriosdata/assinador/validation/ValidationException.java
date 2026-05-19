package com.kyriosdata.assinador.validation;

public final class ValidationException extends RuntimeException {
    private final String field;
    private final String reason;

    public ValidationException(String field, String reason, String message) {
        super(message);
        this.field = field;
        this.reason = reason;
    }

    public String field() {
        return field;
    }

    public String reason() {
        return reason;
    }
}
