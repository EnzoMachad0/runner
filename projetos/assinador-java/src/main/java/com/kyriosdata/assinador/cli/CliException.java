package com.kyriosdata.assinador.cli;

final class CliException extends RuntimeException {
    private final String field;
    private final String reason;

    CliException(String field, String reason, String message) {
        super(message);
        this.field = field;
        this.reason = reason;
    }

    String field() {
        return field;
    }

    String reason() {
        return reason;
    }
}
