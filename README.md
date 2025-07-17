# PilotAI

Zelesonic - AI-Powered Data Analysis

Prerequisites
Before you start, make sure you have Ollama installed and running on your system. This is required to run the local AI models.

Download Ollama 

Installation and Usage
Follow the instructions for your operating system.

On Windows
Download the Windows zip file.

Unzip the downloaded file.

Double-click the zelesonic.exe file to start the application. Go to http://localhost:5000 to use the app.

On macOS
Download the Zelesonic-macOS.zip file.

Unzip the downloaded file.

Open the Terminal app.

Navigate to the unzipped folder using the cd command. Go to http://localhost:5000 to use the app.

# Example:
cd ~/Downloads/Zelesonic-macOS

Make the file executable by running this command:

chmod +x zelesonic-pilot-ai

Start the application by double-clicking the zelesonic file or running it from the terminal:

./zelesonic-pilot-ai

Go to http://localhost:5000 to use the app.

Note: The first time you run it, you may need to go to System Settings > Privacy & Security to allow the application to run.

On Linux
Download the Zelesonic-Linux.zip file.

Unzip the downloaded file.

Open your terminal.

Navigate into the unzipped folder using the cd command.

# Example:
cd ~/Downloads/Zelesonic-Linux

Make the file executable by running this command:

chmod +x zelesonic-pilot-ai

Start the application from the terminal:

./zelesonic-pilot-ai

Go to http://localhost:5000 to use the app.

How to Use the AI Features
Once the application is running, you need to download the necessary AI models through Ollama.

Open your terminal.

Run the following commands one by one to download the models:

ollama pull llama3
ollama pull nomic-embed-text

You can now return to the Zelesonic application, activate the models, upload your files and begin your analysis.

License
This project is licensed under the MIT License. See the LICENSE file for more details.
