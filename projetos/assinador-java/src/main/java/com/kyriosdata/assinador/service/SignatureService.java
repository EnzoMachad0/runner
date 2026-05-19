package com.kyriosdata.assinador.service;

import com.kyriosdata.assinador.domain.SignRequest;
import com.kyriosdata.assinador.domain.SignatureResponse;
import com.kyriosdata.assinador.domain.ValidateRequest;

public interface SignatureService {
    SignatureResponse sign(SignRequest request);

    SignatureResponse validate(ValidateRequest request);
}
