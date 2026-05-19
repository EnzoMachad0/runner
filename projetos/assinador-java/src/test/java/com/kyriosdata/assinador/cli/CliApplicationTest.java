package com.kyriosdata.assinador.cli;

import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.nio.charset.StandardCharsets;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class CliApplicationTest {
    @Test
    void rootHelpListsCommands() {
        CliResult result = run("--help");

        assertEquals(0, result.exitCode());
        assertTrue(result.stdout().contains("sign"));
        assertTrue(result.stdout().contains("validate"));
        assertEquals("", result.stderr());
    }

    @Test
    void signAcceptsKnownFlagsAndPrintsJson() {
        CliResult result = run(
                "sign",
                "--bundle", "{}",
                "--provenance", "{}",
                "--cryptographic-material", "PEM",
                "--certificates", "[]",
                "--reference-timestamp", "1751328000",
                "--timestamp-strategy", "iat",
                "--signature-policy", "uri|v1",
                "--operational-configuration", "{}"
        );

        assertEquals(0, result.exitCode());
        assertTrue(result.stdout().contains("\"success\":true"));
        assertTrue(result.stdout().contains("\"operation\":\"sign\""));
        assertEquals("", result.stderr());
    }

    @Test
    void validateAcceptsKnownFlagsAndPrintsJson() {
        CliResult result = run(
                "validate",
                "--jws", "abc",
                "--reference-timestamp", "1751328000",
                "--signature-policy", "uri|v1",
                "--trust-store", "[]",
                "--revocation-policy", "strict",
                "--ocsp-unknown-handling", "treat-as-revoked"
        );

        assertEquals(0, result.exitCode());
        assertTrue(result.stdout().contains("\"success\":true"));
        assertTrue(result.stdout().contains("\"operation\":\"validate\""));
        assertEquals("", result.stderr());
    }

    @Test
    void unknownCommandReturnsUsageError() {
        CliResult result = run("verify");

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("Comando desconhecido"));
    }

    @Test
    void missingFlagValueReturnsUsageError() {
        CliResult result = run("sign", "--bundle");

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"success\":false"));
        assertTrue(result.stderr().contains("\"operation\":\"sign\""));
        assertTrue(result.stderr().contains("\"field\":\"bundle\""));
        assertTrue(result.stderr().contains("\"reason\":\"missing-value\""));
    }

    @Test
    void signMissingRequiredParameterReturnsValidationError() {
        CliResult result = run("sign", "--bundle", "{}");

        assertEquals(2, result.exitCode());
        assertEquals("", result.stdout());
        assertTrue(result.stderr().contains("\"success\":false"));
        assertTrue(result.stderr().contains("\"operation\":\"sign\""));
        assertTrue(result.stderr().contains("\"reason\":\"required\""));
        assertTrue(result.stderr().contains("--provenance"));
    }

    @Test
    void signInvalidTimestampStrategyReturnsValidationError() {
        CliResult result = run(
                "sign",
                "--bundle", "{}",
                "--provenance", "{}",
                "--cryptographic-material", "PEM",
                "--certificates", "[]",
                "--reference-timestamp", "1751328000",
                "--timestamp-strategy", "local",
                "--signature-policy", "uri|v1",
                "--operational-configuration", "{}"
        );

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"success\":false"));
        assertTrue(result.stderr().contains("\"field\":\"timestamp-strategy\""));
        assertTrue(result.stderr().contains("\"reason\":\"invalid-enum\""));
        assertTrue(result.stderr().contains("--timestamp-strategy"));
        assertTrue(result.stderr().contains("iat"));
        assertTrue(result.stderr().contains("tsa"));
    }

    @Test
    void signInvalidReferenceTimestampReturnsValidationError() {
        CliResult result = run(
                "sign",
                "--bundle", "{}",
                "--provenance", "{}",
                "--cryptographic-material", "PEM",
                "--certificates", "[]",
                "--reference-timestamp", "1",
                "--timestamp-strategy", "iat",
                "--signature-policy", "uri|v1",
                "--operational-configuration", "{}"
        );

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"field\":\"reference-timestamp\""));
        assertTrue(result.stderr().contains("\"reason\":\"out-of-range\""));
        assertTrue(result.stderr().contains("--reference-timestamp"));
        assertTrue(result.stderr().contains("Faixa válida"));
    }

    @Test
    void signInvalidJsonShapeReturnsValidationError() {
        CliResult result = run(
                "sign",
                "--bundle", "texto",
                "--provenance", "{}",
                "--cryptographic-material", "PEM",
                "--certificates", "[]",
                "--reference-timestamp", "1751328000",
                "--timestamp-strategy", "iat",
                "--signature-policy", "uri|v1",
                "--operational-configuration", "{}"
        );

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"field\":\"bundle\""));
        assertTrue(result.stderr().contains("\"reason\":\"invalid-json\""));
        assertTrue(result.stderr().contains("--bundle"));
        assertTrue(result.stderr().contains("JSON"));
    }

    @Test
    void validateMissingRequiredParameterReturnsValidationError() {
        CliResult result = run("validate", "--jws", "abc");

        assertEquals(2, result.exitCode());
        assertEquals("", result.stdout());
        assertTrue(result.stderr().contains("\"success\":false"));
        assertTrue(result.stderr().contains("\"operation\":\"validate\""));
        assertTrue(result.stderr().contains("\"reason\":\"required\""));
        assertTrue(result.stderr().contains("--reference-timestamp"));
    }

    @Test
    void validateInvalidRevocationPolicyReturnsValidationError() {
        CliResult result = run(
                "validate",
                "--jws", "abc",
                "--reference-timestamp", "1751328000",
                "--signature-policy", "uri|v1",
                "--trust-store", "[]",
                "--revocation-policy", "ignore",
                "--ocsp-unknown-handling", "treat-as-revoked"
        );

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"field\":\"revocation-policy\""));
        assertTrue(result.stderr().contains("\"reason\":\"invalid-enum\""));
        assertTrue(result.stderr().contains("--revocation-policy"));
        assertTrue(result.stderr().contains("strict"));
    }

    @Test
    void validateInvalidOptionalNumericRangeReturnsValidationError() {
        CliResult result = run(
                "validate",
                "--jws", "abc",
                "--reference-timestamp", "1751328000",
                "--signature-policy", "uri|v1",
                "--trust-store", "[]",
                "--revocation-policy", "strict",
                "--ocsp-unknown-handling", "treat-as-revoked",
                "--ocsp-crl-tsa-timeout", "1"
        );

        assertEquals(2, result.exitCode());
        assertTrue(result.stderr().contains("\"field\":\"ocsp-crl-tsa-timeout\""));
        assertTrue(result.stderr().contains("\"reason\":\"out-of-range\""));
        assertTrue(result.stderr().contains("--ocsp-crl-tsa-timeout"));
        assertTrue(result.stderr().contains("Faixa válida"));
    }

    private static CliResult run(String... args) {
        ByteArrayOutputStream stdout = new ByteArrayOutputStream();
        ByteArrayOutputStream stderr = new ByteArrayOutputStream();
        int exitCode = new CliApplication(
                new PrintStream(stdout, true, StandardCharsets.UTF_8),
                new PrintStream(stderr, true, StandardCharsets.UTF_8)
        ).run(args);
        return new CliResult(
                exitCode,
                stdout.toString(StandardCharsets.UTF_8),
                stderr.toString(StandardCharsets.UTF_8)
        );
    }

    private record CliResult(int exitCode, String stdout, String stderr) {
    }
}
