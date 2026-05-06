/**
 * Jar mínimo usado nos testes de integração do invoker.
 *
 * Comportamento controlado por flags:
 *   --fail            encerra com código 1 e mensagem de erro no stderr
 *   --sleep <ms>      dorme <ms> milissegundos antes de sair (testa timeout)
 *   --server          inicia servidor HTTP mínimo na porta especificada por --port
 *   --port <N>        porta do servidor (padrão: 8080; requer --server)
 *   qualquer outro    imprime o argumento no stdout
 *
 * O servidor responde HTTP 200 {"status":"ok"} a qualquer requisição.
 * Usa ServerSocket puro (sem APIs internas do JDK) para compatibilidade
 * com qualquer JVM a partir do Java 8.
 */
public class TestJar {
    public static void main(String[] args) throws Exception {
        boolean fail = false;
        boolean server = false;
        long sleepMs = 0;
        int port = 8080;

        for (int i = 0; i < args.length; i++) {
            if ("--fail".equals(args[i])) {
                fail = true;
            } else if ("--sleep".equals(args[i]) && i + 1 < args.length) {
                sleepMs = Long.parseLong(args[i + 1]);
                i++;
            } else if ("--server".equals(args[i])) {
                server = true;
            } else if ("--port".equals(args[i]) && i + 1 < args.length) {
                port = Integer.parseInt(args[i + 1]);
                i++;
            } else {
                System.out.println(args[i]);
            }
        }

        if (sleepMs > 0) {
            Thread.sleep(sleepMs);
        }

        if (fail) {
            System.err.println("erro simulado pelo TestJar");
            System.exit(1);
        }

        if (server) {
            runServer(port);
        }
    }

    /**
     * Servidor HTTP mínimo usando ServerSocket.
     * Aceita qualquer requisição e responde HTTP 200 {"status":"ok"}.
     * Imprime "server:ready:<port>" no stdout quando pronto para receber conexões.
     */
    private static void runServer(int port) throws Exception {
        java.net.ServerSocket serverSocket = new java.net.ServerSocket(port);
        System.out.println("server:ready:" + port);
        System.out.flush();

        while (true) {
            java.net.Socket client;
            try {
                client = serverSocket.accept();
            } catch (java.io.IOException e) {
                if (serverSocket.isClosed()) break;
                continue;
            }
            handleClient(client);
        }
    }

    private static void handleClient(java.net.Socket client) {
        try (client) {
            // Lê e descarta a requisição (não precisamos do conteúdo)
            java.io.InputStream in = client.getInputStream();
            byte[] buf = new byte[4096];
            in.read(buf); // leitura não-bloqueante suficiente para o health check

            String body = "{\"status\":\"ok\"}";
            String response = "HTTP/1.1 200 OK\r\n"
                    + "Content-Type: application/json\r\n"
                    + "Content-Length: " + body.length() + "\r\n"
                    + "Connection: close\r\n"
                    + "\r\n"
                    + body;

            java.io.OutputStream out = client.getOutputStream();
            out.write(response.getBytes("UTF-8"));
            out.flush();
        } catch (Exception ignored) {
            // Conexão encerrada pelo cliente antes da resposta — ignorar
        }
    }
}
