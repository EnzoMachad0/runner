package com.kyriosdata.assinador.domain;

import java.util.List;
import java.util.Map;

public record SignatureResponse(
        boolean success,
        String operation,
        String message,
        Map<String, Object> data,
        List<ValidationError> errors
) {
    public static SignatureResponse success(
            String operation,
            String message,
            Map<String, Object> data
    ) {
        return new SignatureResponse(true, operation, message, data, List.of());
    }

    public static SignatureResponse failure(
            String operation,
            String message,
            String field,
            String reason
    ) {
        return new SignatureResponse(false, operation, message, Map.of(), List.of(new ValidationError(field, reason)));
    }

    public String toJson() {
        StringBuilder json = new StringBuilder();
        json.append("{");
        json.append("\"success\":").append(success).append(",");
        appendField(json, "operation", operation);
        json.append(",");
        appendField(json, "message", message);
        json.append(",");
        json.append("\"data\":");
        appendMap(json, data);
        json.append(",");
        json.append("\"errors\":");
        appendErrors(json, errors);
        json.append("}");
        return json.toString();
    }

    private static void appendMap(StringBuilder json, Map<String, Object> values) {
        json.append("{");
        boolean first = true;
        for (Map.Entry<String, Object> entry : values.entrySet()) {
            if (!first) {
                json.append(",");
            }
            json.append("\"").append(escape(entry.getKey())).append("\":");
            appendValue(json, entry.getValue());
            first = false;
        }
        json.append("}");
    }

    private static void appendErrors(StringBuilder json, List<ValidationError> values) {
        json.append("[");
        boolean first = true;
        for (ValidationError error : values) {
            if (!first) {
                json.append(",");
            }
            json.append("{");
            appendField(json, "field", error.field());
            json.append(",");
            appendField(json, "reason", error.reason());
            json.append("}");
            first = false;
        }
        json.append("]");
    }

    private static void appendValue(StringBuilder json, Object value) {
        if (value instanceof Boolean || value instanceof Number) {
            json.append(value);
            return;
        }
        json.append("\"").append(escape(String.valueOf(value))).append("\"");
    }

    private static void appendField(StringBuilder json, String name, String value) {
        json.append("\"").append(escape(name)).append("\":");
        json.append("\"").append(escape(value)).append("\"");
    }

    private static String escape(String value) {
        return value
                .replace("\\", "\\\\")
                .replace("\"", "\\\"")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t");
    }
}
