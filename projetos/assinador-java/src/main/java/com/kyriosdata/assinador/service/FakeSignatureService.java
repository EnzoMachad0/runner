package com.kyriosdata.assinador.service;

import com.kyriosdata.assinador.domain.SignRequest;
import com.kyriosdata.assinador.domain.SignatureResponse;
import com.kyriosdata.assinador.domain.ValidateRequest;
import com.kyriosdata.assinador.validation.RequestValidator;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;

public final class FakeSignatureService implements SignatureService {
    @Override
    public SignatureResponse sign(SignRequest request) {
        RequestValidator.validateSign(request);

        Map<String, Object> data = new LinkedHashMap<>();
        data.put("signature", "SIMULATED_BASE64_SIGNATURE");
        data.put("algorithm", "SHA256withRSA");
        data.put("signedAt", Instant.now().toString());
        return SignatureResponse.success(
                "sign",
                "Assinatura simulada gerada com sucesso.",
                data
        );
    }

    @Override
    public SignatureResponse validate(ValidateRequest request) {
        RequestValidator.validateValidate(request);

        Map<String, Object> data = new LinkedHashMap<>();
        data.put("valid", true);
        data.put("signer", "Certificado Simulado");
        data.put("checkedAt", Instant.now().toString());
        return SignatureResponse.success(
                "validate",
                "Assinatura válida.",
                data
        );
    }
}
