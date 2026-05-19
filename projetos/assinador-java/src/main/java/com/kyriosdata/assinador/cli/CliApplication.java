package com.kyriosdata.assinador.cli;

import com.kyriosdata.assinador.domain.SignRequest;
import com.kyriosdata.assinador.domain.SignatureResponse;
import com.kyriosdata.assinador.domain.ValidateRequest;
import com.kyriosdata.assinador.service.FakeSignatureService;
import com.kyriosdata.assinador.service.SignatureService;
import com.kyriosdata.assinador.validation.ValidationException;

import java.io.PrintStream;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

public final class CliApplication {
    private static final Set<String> SIGN_FLAGS = Set.of(
            "bundle",
            "provenance",
            "cryptographic-material",
            "certificates",
            "reference-timestamp",
            "timestamp-strategy",
            "signature-policy",
            "operational-configuration"
    );

    private static final Set<String> VALIDATE_FLAGS = Set.of(
            "jws",
            "reference-timestamp",
            "signature-policy",
            "trust-store",
            "revocation-policy",
            "ocsp-unknown-handling",
            "min-cert-issue-date",
            "ocsp-crl-tsa-timeout",
            "revocation-cache-ttl",
            "near-expiry-threshold-days",
            "signature-age-threshold-days",
            "original-bundle",
            "original-provenance",
            "max-entries-bundle",
            "max-bundle-bytes",
            "bundle-verify-timeout"
    );

    private final PrintStream out;
    private final PrintStream err;
    private final SignatureService signatureService;

    public CliApplication(PrintStream out, PrintStream err) {
        this(out, err, new FakeSignatureService());
    }

    CliApplication(PrintStream out, PrintStream err, SignatureService signatureService) {
        this.out = out;
        this.err = err;
        this.signatureService = signatureService;
    }

    public int run(String[] args) {
        if (args.length == 0) {
            err.println(SignatureResponse.failure(
                    "unknown",
                    "Comando obrigatório ausente: use sign ou validate.",
                    "command",
                    "required"
            ).toJson());
            return 2;
        }

        String command = args[0];
        if (isHelp(command)) {
            printRootHelp(out);
            return 0;
        }

        return switch (command) {
            case "sign" -> runSign(args);
            case "validate" -> runValidate(args);
            default -> {
                err.println(SignatureResponse.failure(
                        "unknown",
                        "Comando desconhecido: " + command + ". Comandos aceitos: sign, validate.",
                        "command",
                        "unknown-command"
                ).toJson());
                yield 2;
            }
        };
    }

    private int runSign(String[] args) {
        if (args.length == 2 && isHelp(args[1])) {
            printSignHelp(out);
            return 0;
        }

        try {
            Map<String, String> options = parseOptions(args, SIGN_FLAGS);
            SignRequest request = new SignRequest(
                    options.get("bundle"),
                    options.get("provenance"),
                    options.get("cryptographic-material"),
                    options.get("certificates"),
                    options.get("reference-timestamp"),
                    options.get("timestamp-strategy"),
                    options.get("signature-policy"),
                    options.get("operational-configuration")
            );
            out.println(signatureService.sign(request).toJson());
            return 0;
        } catch (CliException e) {
            err.println(SignatureResponse.failure("sign", e.getMessage(), e.field(), e.reason()).toJson());
            return 2;
        } catch (ValidationException e) {
            err.println(SignatureResponse.failure("sign", e.getMessage(), e.field(), e.reason()).toJson());
            return 2;
        }
    }

    private int runValidate(String[] args) {
        if (args.length == 2 && isHelp(args[1])) {
            printValidateHelp(out);
            return 0;
        }

        try {
            Map<String, String> options = parseOptions(args, VALIDATE_FLAGS);
            ValidateRequest request = new ValidateRequest(options);
            out.println(signatureService.validate(request).toJson());
            return 0;
        } catch (CliException e) {
            err.println(SignatureResponse.failure("validate", e.getMessage(), e.field(), e.reason()).toJson());
            return 2;
        } catch (ValidationException e) {
            err.println(SignatureResponse.failure("validate", e.getMessage(), e.field(), e.reason()).toJson());
            return 2;
        }
    }

    private static Map<String, String> parseOptions(String[] args, Set<String> allowedFlags) {
        Map<String, String> options = new LinkedHashMap<>();
        for (int i = 1; i < args.length; i += 2) {
            String rawFlag = args[i];
            if (!rawFlag.startsWith("--") || rawFlag.length() == 2) {
                throw new CliException(
                        "argument",
                        "invalid-flag",
                        "Flag inválida: " + rawFlag + ". Use o formato --nome valor."
                );
            }

            String flag = rawFlag.substring(2);
            if (!allowedFlags.contains(flag)) {
                throw new CliException(
                        flag,
                        "unknown-flag",
                        "Flag desconhecida para este comando: --" + flag
                );
            }

            if (i + 1 >= args.length) {
                throw new CliException(
                        flag,
                        "missing-value",
                        "Valor ausente para a flag --" + flag + "."
                );
            }

            String value = args[i + 1];
            if (value.startsWith("--")) {
                throw new CliException(
                        flag,
                        "missing-value",
                        "Valor ausente para a flag --" + flag + "."
                );
            }

            options.put(flag, value);
        }
        return options;
    }

    private static boolean isHelp(String value) {
        return "--help".equals(value) || "-h".equals(value);
    }

    private static void printRootHelp(PrintStream out) {
        out.println("Uso: java -jar assinador.jar <comando> [flags]");
        out.println();
        out.println("Comandos:");
        out.println("  sign      Cria uma assinatura simulada para Bundle FHIR");
        out.println("  validate  Valida uma assinatura JWS simulada");
        out.println();
        out.println("Use java -jar assinador.jar <comando> --help para detalhes.");
    }

    private static void printSignHelp(PrintStream out) {
        out.println("Uso: java -jar assinador.jar sign [flags]");
        out.println();
        out.println("Flags aceitas:");
        SIGN_FLAGS.stream().sorted().forEach(flag -> out.println("  --" + flag + " <valor>"));
    }

    private static void printValidateHelp(PrintStream out) {
        out.println("Uso: java -jar assinador.jar validate [flags]");
        out.println();
        out.println("Flags aceitas:");
        VALIDATE_FLAGS.stream().sorted().forEach(flag -> out.println("  --" + flag + " <valor>"));
    }
}
