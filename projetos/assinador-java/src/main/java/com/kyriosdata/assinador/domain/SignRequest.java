package com.kyriosdata.assinador.domain;

public record SignRequest(
        String bundle,
        String provenance,
        String cryptographicMaterial,
        String certificates,
        String referenceTimestamp,
        String timestampStrategy,
        String signaturePolicy,
        String operationalConfiguration
) {
}
