# PilotAI
Local AI Data Analysis with Ollama &amp; Executable Python

Prerequisites
Before you begin, ensure you have the following software installed on your system.

Python: Version 3.8 or higher.

Pip: Python's package installer (usually comes with Python).

Git: For cloning the repository.

Ollama: To run local large language models. Download here.

MATLAB (Optional): Required for specific advanced mathematical or engineering-related analysis modules. The core application will function without it.

1. Installation
Follow these steps to get your local development environment set up.

Clone the Repository
First, clone the repository to your local machine using Git:

git clone https://github.com/your-username/zelesonic-site.git
cd zelesonic-site

Create a Virtual Environment & Install Dependencies
It is highly recommended to use a Python virtual environment to manage project dependencies.

On macOS / Linux:

# Create a virtual environment
python3 -m venv venv

# Activate the virtual environment
source venv/bin/activate

# Install the required packages
pip install -r requirements.txt

On Windows:

# Create a virtual environment
python -m venv venv

# Activate the virtual environment
.\venv\Scripts\activate

# Install the required packages
pip install -r requirements.txt

(Note: If you do not have a requirements.txt file yet, you will need to create one by running pip freeze > requirements.txt after installing your project's dependencies.)

2. Configuration: Setting Up Local AI Models
This application uses Ollama to run AI models on your own machine.

Step 1: Install and Run Ollama
If you haven't already, download and install Ollama. After installation, ensure the Ollama application is running. You can verify it's working by opening your terminal and running:

ollama --version

Step 2: Download the AI Models
We need one generative model for text-based analysis and one embedding model for understanding data context. We recommend the following models, which you can pull using these commands:

# Pull the generative model (e.g., Llama 3)
ollama pull llama3

# Pull the embedding model (e.g., Nomic Embed Text)
ollama pull nomic-embed-text

You can choose other models from the Ollama library if you prefer.

Step 3: Configure the Application
The application needs to know which models to use.

Find the .env.example file in the project root.

Create a copy of it and name it .env.

Open the new .env file and set the model names to match the ones you downloaded:

# .env file
GENERATIVE_MODEL=llama3
EMBEDDING_MODEL=nomic-embed-text

The application will load these settings on startup.

3. Usage
With everything installed and configured, you are ready to run the application.

Step 1: Activate the Environment
Make sure your Python virtual environment is still active. If not, reactivate it:

macOS/Linux: source venv/bin/activate

Windows: .\venv\Scripts\activate

Step 2: Run the Application
Start the main application script (e.g., app.py):

python app.py

Step 3: Analyze Your Data
Once the application is running:

Upload Your Files: Use the interface to upload your data files (e.g., .csv, .json, .txt).

Ask for Analysis: Interact with the application by typing requests in natural language. The AI will use the configured models to process your request.

Example Prompts:

"Provide a statistical summary of the sales_data.csv file."

"What is the correlation between column A and column B?"

"Generate a bar chart showing total revenue per month."

"Identify any outliers in the user activity log."

License
This project is licensed under the MIT License. See the LICENSE file for more details.
