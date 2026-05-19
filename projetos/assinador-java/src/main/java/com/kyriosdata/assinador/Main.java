package com.kyriosdata.assinador;

import com.kyriosdata.assinador.cli.CliApplication;

public final class Main {
    private Main() {
    }

    public static void main(String[] args) {
        int exitCode = new CliApplication(System.out, System.err).run(args);
        if (exitCode != 0) {
            System.exit(exitCode);
        }
    }
}
