package com.kyriosdata.assinador.validation;

import com.kyriosdata.assinador.domain.SignRequest;
import com.kyriosdata.assinador.domain.ValidateRequest;

import java.util.List;
import java.util.Map;

public final class RequestValidator {
    private static final long REFERENCE_TIMESTAMP_MIN = 1_751_328_000L;
    private static final long REFERENCE_TIMESTAMP_MAX = 4_102_444_800L;
    private static final long MIN_CERT_ISSUE_DATE_MIN = 1_609_459_200L;

    private static final List<String> TIMESTAMP_STRATEGIES = List.of("iat", "tsa");
    private static final List<String> CRYPTOGRAPHIC_MATERIALS = List.of(
            "PEM",
            "PKCS#12",
            "SMARTCARD",
            "TOKEN",
            "REMOTE"
    );
    private static final List<String> REVOCATION_POLICIES = List.of("strict", "soft-fail", "warn");
    private static final List<String> OCSP_UNKNOWN_HANDLINGS = List.of(
            "treat-as-revoked",
            "treat-as-warning"
    );

    private RequestValidator() {
    }

    public static void validateSign(SignRequest request) {
        require(request.bundle(), "bundle");
        require(request.provenance(), "provenance");
        require(request.cryptographicMaterial(), "cryptographic-material");
        require(request.certificates(), "certificates");
        require(request.referenceTimestamp(), "reference-timestamp");
        require(request.timestampStrategy(), "timestamp-strategy");
        require(request.signaturePolicy(), "signature-policy");
        require(request.operationalConfiguration(), "operational-configuration");

        requireJsonObjectOrArray(request.bundle(), "bundle");
        requireJsonObject(request.provenance(), "provenance");
        requireJsonObjectOrArray(request.certificates(), "certificates");
        requireJsonObject(request.operationalConfiguration(), "operational-configuration");
        requireOneOf(request.cryptographicMaterial(), "cryptographic-material", CRYPTOGRAPHIC_MATERIALS);
        requireLongInRange(
                request.referenceTimestamp(),
                "reference-timestamp",
                REFERENCE_TIMESTAMP_MIN,
                REFERENCE_TIMESTAMP_MAX
        );
        requireOneOf(request.timestampStrategy(), "timestamp-strategy", TIMESTAMP_STRATEGIES);
        requireSignaturePolicy(request.signaturePolicy());
    }

    public static void validateValidate(ValidateRequest request) {
        Map<String, String> options = request.options();

        require(options.get("jws"), "jws");
        require(options.get("reference-timestamp"), "reference-timestamp");
        require(options.get("signature-policy"), "signature-policy");
        require(options.get("trust-store"), "trust-store");
        require(options.get("revocation-policy"), "revocation-policy");
        require(options.get("ocsp-unknown-handling"), "ocsp-unknown-handling");

        requireLongInRange(
                options.get("reference-timestamp"),
                "reference-timestamp",
                REFERENCE_TIMESTAMP_MIN,
                REFERENCE_TIMESTAMP_MAX
        );
        requireSignaturePolicy(options.get("signature-policy"));
        requireJsonObjectOrArray(options.get("trust-store"), "trust-store");
        requireOneOf(options.get("revocation-policy"), "revocation-policy", REVOCATION_POLICIES);
        requireOneOf(options.get("ocsp-unknown-handling"), "ocsp-unknown-handling", OCSP_UNKNOWN_HANDLINGS);

        validateOptionalLong(options, "min-cert-issue-date", MIN_CERT_ISSUE_DATE_MIN, REFERENCE_TIMESTAMP_MAX);
        validateOptionalInt(options, "ocsp-crl-tsa-timeout", 5, 120);
        validateOptionalInt(options, "revocation-cache-ttl", 300, 86_400);
        validateOptionalInt(options, "near-expiry-threshold-days", 1, 180);
        validateOptionalInt(options, "signature-age-threshold-days", 1, 1_825);
        validateOptionalPositiveInt(options, "max-entries-bundle");
        validateOptionalPositiveInt(options, "max-bundle-bytes");
        validateOptionalPositiveInt(options, "bundle-verify-timeout");
        validateOptionalJsonObject(options, "original-provenance");
        validateOptionalJsonObjectOrArray(options, "original-bundle");
    }

    private static void validateOptionalLong(
            Map<String, String> options,
            String field,
            long min,
            long max
    ) {
        String value = options.get(field);
        if (isBlank(value)) {
            return;
        }
        requireLongInRange(value, field, min, max);
    }

    private static void validateOptionalInt(Map<String, String> options, String field, int min, int max) {
        String value = options.get(field);
        if (isBlank(value)) {
            return;
        }
        requireIntInRange(value, field, min, max);
    }

    private static void validateOptionalPositiveInt(Map<String, String> options, String field) {
        String value = options.get(field);
        if (isBlank(value)) {
            return;
        }
        requireIntInRange(value, field, 1, Integer.MAX_VALUE);
    }

    private static void validateOptionalJsonObject(Map<String, String> options, String field) {
        String value = options.get(field);
        if (isBlank(value)) {
            return;
        }
        requireJsonObject(value, field);
    }

    private static void validateOptionalJsonObjectOrArray(Map<String, String> options, String field) {
        String value = options.get(field);
        if (isBlank(value)) {
            return;
        }
        requireJsonObjectOrArray(value, field);
    }

    private static void require(String value, String field) {
        if (isBlank(value)) {
            throw new ValidationException(
                    field,
                    "required",
                    "Parâmetro obrigatório ausente ou vazio: --" + field + "."
            );
        }
    }

    private static void requireOneOf(String value, String field, List<String> allowedValues) {
        if (!allowedValues.contains(value)) {
            throw new ValidationException(
                    field,
                    "invalid-enum",
                    "Valor inválido para --" + field + ": " + quote(value)
                            + ". Valores aceitos: " + String.join(", ", allowedValues) + "."
            );
        }
    }

    private static void requireSignaturePolicy(String value) {
        String[] parts = value.split("\\|", -1);
        if (parts.length != 2 || isBlank(parts[0]) || isBlank(parts[1])) {
            throw new ValidationException(
                    "signature-policy",
                    "invalid-format",
                    "Valor inválido para --signature-policy: use o formato {baseUri}|{versão}."
            );
        }
    }

    private static void requireJsonObject(String value, String field) {
        String trimmed = value.trim();
        if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) {
            throw new ValidationException(
                    field,
                    "invalid-json",
                    "Valor inválido para --" + field + ": informe um JSON object."
            );
        }
    }

    private static void requireJsonObjectOrArray(String value, String field) {
        String trimmed = value.trim();
        boolean object = trimmed.startsWith("{") && trimmed.endsWith("}");
        boolean array = trimmed.startsWith("[") && trimmed.endsWith("]");
        if (!object && !array) {
            throw new ValidationException(
                    field,
                    "invalid-json",
                    "Valor inválido para --" + field + ": informe um JSON object ou array."
            );
        }
    }

    private static void requireLongInRange(String value, String field, long min, long max) {
        long parsed;
        try {
            parsed = Long.parseLong(value);
        } catch (NumberFormatException e) {
            throw new ValidationException(
                    field,
                    "invalid-integer",
                    "Valor inválido para --" + field + ": informe um inteiro."
            );
        }
        if (parsed < min || parsed > max) {
            throw new ValidationException(
                    field,
                    "out-of-range",
                    "Valor inválido para --" + field + ": " + parsed
                            + ". Faixa válida: [" + min + ", " + max + "]."
            );
        }
    }

    private static void requireIntInRange(String value, String field, int min, int max) {
        int parsed;
        try {
            parsed = Integer.parseInt(value);
        } catch (NumberFormatException e) {
            throw new ValidationException(
                    field,
                    "invalid-integer",
                    "Valor inválido para --" + field + ": informe um inteiro."
            );
        }
        if (parsed < min || parsed > max) {
            throw new ValidationException(
                    field,
                    "out-of-range",
                    "Valor inválido para --" + field + ": " + parsed
                            + ". Faixa válida: [" + min + ", " + max + "]."
            );
        }
    }

    private static boolean isBlank(String value) {
        return value == null || value.trim().isEmpty();
    }

    private static String quote(String value) {
        if (value == null) {
            return "\"\"";
        }
        return "\"" + value + "\"";
    }
}
