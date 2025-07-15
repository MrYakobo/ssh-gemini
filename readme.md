# Gemini 1.5 Flash over SSH

have you ever found yourself on a machine with only an terminal? no web browser?
no one? eh, whatever. my firefox couldn't start and i needed to google some stuff.

![](./demo.svg)

## remote usage

```bash
# this service won't be up for very long ;)
ssh -p 2222 gemini.lind.sk

# Gemini 1.5 Flash over SSH

Prompt: hello who are you?

   Thinking...

Response:
I am a large language model, trained by Google.  I'm here to help answe
```
## building locally

*   Go >= 1.18
*   A valid API key for the Gemini API in GEMINI_API_KEY

## building

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/your-username/gemini-ssh.git
    cd gemini-ssh
    ```

2.  **Set the `GEMINI_API_KEY` environment variable:**

    ```bash
    export GEMINI_API_KEY="YOUR_API_KEY"
    ```

    Replace `YOUR_API_KEY` with your actual Gemini API key.  It's best practice to set this in your `.bashrc`, `.zshrc`, or other shell configuration file.

3.  **Install dependencies:**

    ```bash
    go mod tidy
    ```

4.  **Build the application:**

    ```bash
    go build .
    ```

## usage

0. **Generate ssh keys**
    The server expects a key named `id_ed25519` in the current directory. You can generate such key using `ssh-keygen -t ed25519 -f id_ed25519`

1.  **Run the server:**

    ```bash
    ./gemini-ssh
    ```

    This will start the SSH server on port 2222 (by default).

2.  **Connect to the server using an SSH client:**

    ```bash
    ssh -p 2222 localhost
    ```

    **Note:** You might need to accept the host key fingerprint the first time you connect.

## configuration

*   **API Key:** The `GEMINI_API_KEY` environment variable must be set for the application to work.
*   **Port:** The SSH server listens on the port given by `GEMINI_PORT` variable, or 2222 by default.

## quick testing

A simple test mode can be activated by setting the `TEST` environment variable. This runs a basic query to Gemini and prints the result to the console:

```bash
TEST=true ./gemini-ssh
```

This is useful for verifying that the API key is configured correctly and that the application can communicate with the Gemini API.

## License

MIT
