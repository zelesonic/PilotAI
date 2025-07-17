// app.js - Open Source Version

// --- Global DOM Elements ---
let themeToggle;
let appPanel;
let chatBoxView;
let rightPanel, rightPanelTitle, rightPanelContent, closePanelBtn;
let uploadView, aiConfigView;
let fileUploadInput, uploadButton, uploadStatus;
let chatInputField, sendChatBtn, chatMessagesDiv;
let baseUrlInput, embeddingModelInput, generativeModelInput, saveModelSettingsBtn;
let activateEmbeddingSelect, activateGenerativeSelect, activateModelsBtn;
let activeEmbeddingModelSpan, activeGenerativeModelSpan, savedModelsList, ollamaStatus;
let newChatBtn, uploadFilesBtn, configureAiBtn;
let documentList;

// --- API Helper Function ---
/**
 * A simplified API helper for making requests to the local backend.
 * @param {string} endpoint - The API endpoint to call (e.g., '/api/chat').
 * @param {string} [method='GET'] - The HTTP method to use.
 * @param {object|null} [data=null] - The JSON data to send in the request body.
 * @returns {Promise<object>} - A promise that resolves to the JSON response from the backend.
 */
async function callBackendApi(endpoint, method = 'GET', data = null) {
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        }
    };
    if (data) {
        options.body = JSON.stringify(data);
    }
    try {
        const response = await fetch(endpoint, options);
        if (!response.ok) {
            const responseData = await response.json().catch(() => ({ error: `API call failed with status ${response.status}` }));
            console.error("API Error:", responseData);
            return { error: responseData.error || `API call failed with status ${response.status}` };
        }
        return await response.json();
    } catch (error) {
        console.error("Network error during API call:", error);
        return { error: `Network error: ${error.message}` };
    }
}

// --- DOM Ready & Event Listeners ---
document.addEventListener('DOMContentLoaded', () => {

    // --- Chat Persistence ---
    function saveChatState() {
        localStorage.setItem('chatHTML', chatMessagesDiv.innerHTML);
    }

    function loadChatState() {
        const savedChat = localStorage.getItem('chatHTML');
        if (savedChat) {
            chatMessagesDiv.innerHTML = savedChat;
        }
    }

    // --- Assign DOM Elements ---
    themeToggle = document.getElementById('theme-toggle');
    appPanel = document.getElementById('app-panel');
    newChatBtn = document.getElementById('new-chat-btn');
    uploadFilesBtn = document.getElementById('upload-files-btn');
    configureAiBtn = document.getElementById('configure-ai-btn');
    chatBoxView = document.getElementById('chat-box-view');
    rightPanel = document.getElementById('right-panel');
    rightPanelTitle = document.getElementById('right-panel-title');
    rightPanelContent = document.getElementById('right-panel-content');
    closePanelBtn = document.querySelector('.close-panel-btn');
    uploadView = document.getElementById('upload-view');
    aiConfigView = document.getElementById('ai-config-view');
    fileUploadInput = document.getElementById('file-upload-input');
    uploadButton = document.getElementById('upload-button');
    uploadStatus = document.getElementById('upload-status');
    chatInputField = document.getElementById('chat-input-field');
    sendChatBtn = document.getElementById('send-chat-btn');
    chatMessagesDiv = document.querySelector('.chat-messages');
    baseUrlInput = document.getElementById('base-url-input');
    embeddingModelInput = document.getElementById('embedding-model-input');
    generativeModelInput = document.getElementById('generative-model-input');
    saveModelSettingsBtn = document.getElementById('save-model-settings-btn');
    activateEmbeddingSelect = document.getElementById('activate-embedding-select');
    activateGenerativeSelect = document.getElementById('activate-generative-select');
    activateModelsBtn = document.getElementById('activate-models-btn');
    activeEmbeddingModelSpan = document.getElementById('active-embedding-model');
    activeGenerativeModelSpan = document.getElementById('active-generative-model');
    savedModelsList = document.getElementById('saved-models-list');
    ollamaStatus = document.getElementById('ollama-status');
    documentList = document.getElementById('document-list');

    const rightPanelViews = {
        'upload': uploadView,
        'ai-config': aiConfigView,
    };

    function showRightPanel(panelId, titleText) {
        while (rightPanelContent.firstChild) {
            rightPanelContent.removeChild(rightPanelContent.firstChild);
        }
        const viewToShow = rightPanelViews[panelId];
        if (viewToShow) {
            rightPanelTitle.textContent = titleText;
            rightPanelContent.appendChild(viewToShow);
            viewToShow.style.display = 'flex';
            rightPanel.style.display = 'flex';
        }
    }

    function hideRightPanel() {
        if (!rightPanel) return;
        rightPanel.style.display = 'none';
        const currentView = rightPanelContent.querySelector('.content-view');
        if (currentView) {
            currentView.style.display = 'none';
            document.querySelector('.main-content').appendChild(currentView);
        }
    }

    function addCodeBlockToChat(code) {
        const codeBlock = document.createElement('div');
        codeBlock.className = 'code-block';

        const editor = document.createElement('textarea');
        editor.className = 'code-editor';
        editor.value = code;

        const controls = document.createElement('div');
        controls.className = 'code-block-controls';

        const runButton = document.createElement('button');
        runButton.className = 'run-button';
        runButton.textContent = 'Run';

        const outputArea = document.createElement('div');
        outputArea.className = 'code-output';
        outputArea.textContent = 'Click "Run" to execute the code above.';

        controls.appendChild(runButton);
        codeBlock.appendChild(editor);
        codeBlock.appendChild(controls);
        codeBlock.appendChild(outputArea);

        const messageWrapper = addMessageToChat('ai', '');
        messageWrapper.classList.add('has-code-block');
        const messageContent = messageWrapper.querySelector('.message-content');
        messageContent.innerHTML = '';
        messageContent.appendChild(codeBlock);

        runButton.addEventListener('click', async () => {
            runButton.disabled = true;
            runButton.textContent = 'Running...';
            outputArea.classList.remove('error');
            outputArea.innerHTML = 'Executing...';

            const codeToRun = editor.value;
            const response = await callBackendApi('/api/execute', 'POST', { code: codeToRun });

            outputArea.innerHTML = '';

            if (response.error) {
                outputArea.textContent = `Execution Failed:\n${response.error}`;
                outputArea.classList.add('error');
            } else {
                if (response.chart) {
                    const chartImg = document.createElement('img');
                    chartImg.src = response.chart;
                    chartImg.style.maxWidth = '100%';
                    chartImg.style.borderRadius = '8px';
                    chartImg.style.marginTop = '10px';
                    outputArea.appendChild(chartImg);
                }
                if (response.result && response.result.trim() !== "") {
                    const textResult = document.createElement('pre');
                    textResult.textContent = response.result;
                    outputArea.appendChild(textResult);
                }
                if (!response.chart && (!response.result || response.result.trim() === "")) {
                    outputArea.textContent = '(No output)';
                }
            }

            runButton.disabled = false;
            runButton.textContent = 'Run';
        });
    }

    async function updateOllamaConfigUI() {
        const statusResponse = await callBackendApi('/api/ollama/status');
        if (statusResponse.error) {
            ollamaStatus.textContent = `Error: ${statusResponse.error}`;
            return;
        }
        ollamaStatus.textContent = statusResponse.message;
        activeEmbeddingModelSpan.textContent = statusResponse.active_embedding_model || 'None';
        activeGenerativeModelSpan.textContent = statusResponse.active_generative_model || 'None';
        baseUrlInput.value = statusResponse.current_base_url || '';

        const configResponse = await callBackendApi('/api/ollama/config', 'GET');
        if (configResponse.error) {
            savedModelsList.innerHTML = `<p class="placeholder-text-panel">Error loading: ${configResponse.error}</p>`;
            return;
        }
        savedModelsList.innerHTML = '';
        const allSaved = [...(configResponse.saved_models.embedding || []), ...(configResponse.saved_models.generative || [])];
        if (allSaved.length === 0) {
            savedModelsList.innerHTML = '<p class="placeholder-text-panel">No models saved yet.</p>';
        } else {
            allSaved.forEach(modelName => {
                const modelType = configResponse.saved_models.embedding.includes(modelName) ? 'embedding' : 'generative';
                const listItem = document.createElement('li');
                listItem.innerHTML = `<span>${modelName} (${modelType})</span> <button class="delete-model-btn" data-model-name="${modelName}" data-model-type="${modelType}">Delete</button>`;
                savedModelsList.appendChild(listItem);
            });
        }

        savedModelsList.querySelectorAll('.delete-model-btn').forEach(button => {
            button.addEventListener('click', async () => {
                const modelName = button.dataset.modelName;
                const modelType = button.dataset.modelType;
                if (confirm(`Are you sure you want to delete ${modelName}?`)) {
                    await callBackendApi('/api/ollama/models/delete', 'POST', { model_name: modelName, model_type: modelType });
                    updateOllamaConfigUI();
                }
            });
        });

        const savedEmbeddingModels = configResponse.saved_models.embedding || [];
        activateEmbeddingSelect.innerHTML = '<option value="">Select an embedding model</option>';
        savedEmbeddingModels.forEach(modelName => {
            const option = document.createElement('option');
            option.value = modelName;
            option.textContent = modelName;
            activateEmbeddingSelect.appendChild(option);
        });
        if (statusResponse.active_embedding_model) {
            activateEmbeddingSelect.value = statusResponse.active_embedding_model;
        }

        const savedGenerativeModels = configResponse.saved_models.generative || [];
        activateGenerativeSelect.innerHTML = '<option value="">Select a generative model</option>';
        savedGenerativeModels.forEach(modelName => {
            const option = document.createElement('option');
            option.value = modelName;
            option.textContent = modelName;
            activateGenerativeSelect.appendChild(option);
        });
        if (statusResponse.active_generative_model) {
            activateGenerativeSelect.value = statusResponse.active_generative_model;
        }
    }

    function addMessageToChat(sender, text) {
        const messageWrapper = document.createElement('div');
        messageWrapper.classList.add('message-wrapper', sender === 'user' ? 'user' : 'ai');

        const avatar = document.createElement('div');
        avatar.classList.add('avatar');
        avatar.textContent = sender === 'user' ? 'You' : 'AI';

        const messageContent = document.createElement('div');
        messageContent.classList.add('message-content');

        if (sender === 'user') {
            messageContent.innerText = text;
        } else {
            if (text) {
                messageContent.innerHTML = marked.parse(text, { sanitize: true });
            }
        }

        if (sender === 'user') {
            messageWrapper.appendChild(messageContent);
            messageWrapper.appendChild(avatar);
        } else {
            messageWrapper.appendChild(avatar);
            messageWrapper.appendChild(messageContent);
        }

        chatMessagesDiv.appendChild(messageWrapper);
        messageWrapper.scrollIntoView({ behavior: 'smooth', block: 'end' });
        return messageWrapper;
    }

    // --- Theme Handling ---
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'light') {
        document.body.classList.add('light-theme');
        themeToggle.checked = false;
    } else {
        document.body.classList.remove('light-theme');
        themeToggle.checked = true;
    }

    themeToggle.addEventListener('change', () => {
        document.body.classList.toggle('light-theme', !themeToggle.checked);
        localStorage.setItem('theme', themeToggle.checked ? 'dark' : 'light');
    });

    // --- Core Application Event Listeners ---
    newChatBtn.addEventListener('click', (e) => {
        e.preventDefault();
        chatMessagesDiv.innerHTML = '<p class="ai-message">To begin, please go to "Artificial Intelligence (AI)" to add and activate your models, then go to "Upload & View Files" to upload and select a document.</p>';
        chatInputField.value = '';
        localStorage.removeItem('chatHTML');
    });

    uploadFilesBtn.addEventListener('click', async (e) => {
        e.preventDefault();
        showRightPanel('upload', 'Upload & Manage Documents');
        await updateDocumentList();
    });

    configureAiBtn.addEventListener('click', (e) => {
        e.preventDefault();
        showRightPanel('ai-config', 'Configure AI Settings');
        updateOllamaConfigUI();
    });

    closePanelBtn.addEventListener('click', hideRightPanel);

    // --- File Upload Logic ---
    fileUploadInput.addEventListener('change', async (e) => {
        const file = e.target.files[0];
        if (file) {
            uploadButton.disabled = true;
            uploadButton.textContent = 'Processing...';
            uploadStatus.textContent = `Uploading "${file.name}"...`;
            const formData = new FormData();
            formData.append('file', file);
            
            // The fetch call no longer requires an Authorization header.
            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData,
            });

            const result = await response.json();
            if (!response.ok || result.status === 'failed') {
                uploadStatus.textContent = `Error: ${result.error || 'Upload failed'}`;
            } else {
                uploadStatus.textContent = `Upload successful. Starting background processing...`;
                const newDocumentId = result.documentId;
                if (newDocumentId) {
                    await callBackendApi('/api/documents/select', 'POST', { id: newDocumentId });
                    await new Promise(resolve => setTimeout(resolve, 200));
                    await updateDocumentList();
                } else {
                    await updateDocumentList();
                }
            }
            e.target.value = ''; // Clear file input
        }
    });

    async function updateDocumentList() {
        const response = await callBackendApi('/api/documents');
        const currentFileDisplay = document.getElementById('current-file-display');
        const currentFileNameSpan = document.getElementById('current-file-name');

        if (!response || response.error) {
            documentList.innerHTML = `<p class="placeholder-text-panel">Error: ${response ? response.error : 'Could not load documents.'}</p>`;
            return;
        }

        const { documents, activeDocumentID } = response;
        documentList.innerHTML = '';
        let isProcessing = false;

        if (documents && documents.length > 0) {
            let activeDoc = null;
            documents.forEach(doc => {
                const listItem = document.createElement('li');
                listItem.dataset.docId = doc.id;
                
                if (doc.id === activeDocumentID) {
                    listItem.classList.add('selected');
                    activeDoc = doc;
                }

                let statusIndicator = '';
                if (doc.status === 'processing') {
                    isProcessing = true;
                    statusIndicator = ` <span class="processing-indicator">(${doc.processingProgress || 'Processing...'})</span>`;
                } else if (doc.status === 'failed') {
                    statusIndicator = ` <span class="failed-indicator">(Failed: ${doc.processingProgress})</span>`;
                }

                listItem.innerHTML = `<span>${doc.fileName}</span>${statusIndicator}<button class="delete-file-btn" data-doc-id="${doc.id}">&times;</button>`;
                documentList.appendChild(listItem);
            });
            
            if (activeDoc) {
                currentFileNameSpan.textContent = activeDoc.fileName;
                currentFileDisplay.style.display = 'block';
            } else {
                currentFileDisplay.style.display = 'none';
            }

        } else {
            documentList.innerHTML = '<p class="placeholder-text-panel">No documents uploaded yet.</p>';
            currentFileDisplay.style.display = 'none';
        }

        uploadButton.disabled = isProcessing;

        if (isProcessing) {
            uploadButton.textContent = 'Processing...';
            setTimeout(updateDocumentList, 2000);
        } else {
            uploadButton.textContent = 'Choose File to Upload';
        }
    }
    
    uploadButton.addEventListener('click', () => fileUploadInput.click());

    documentList.addEventListener('click', async (e) => {
        const listItem = e.target.closest('li');
        if (!listItem) return;

        if (e.target.classList.contains('delete-file-btn')) {
            e.stopPropagation();
            const button = e.target;
            const docId = button.dataset.docId;
            const fileName = button.previousElementSibling.textContent;
            if (confirm(`Are you sure you want to delete "${fileName}"?`)) {
                button.disabled = true;
                const response = await callBackendApi('/api/documents/delete', 'POST', { id: docId });
                if (response.error) {
                    alert(`Failed to delete: ${response.error}`);
                    button.disabled = false;
                } else {
                    await updateDocumentList();
                }
            }
            return;
        }
        
        const docId = listItem.dataset.docId;
        const response = await callBackendApi('/api/documents/select', 'POST', { id: docId });
        if (response.error) {
            alert(`Failed to select file: ${response.error}`);
        } else {
            await updateDocumentList();
        }
    });

    // --- Chat Message Logic ---
    const sendChatMessage = async () => {
        const message = chatInputField.value.trim();
        if (!message) return;

        addMessageToChat('user', message);
        chatInputField.value = '';
        chatInputField.disabled = true;
        sendChatBtn.disabled = true;

        addMessageToChat('ai', '<div class="thinking"><span>.</span><span>.</span><span>.</span></div>');

        try {
            const response = await callBackendApi('/api/chat', 'POST', { prompt: message });
            
            const thinkingBubble = document.querySelector('.message-content .thinking');
            if (thinkingBubble) {
                thinkingBubble.closest('.message-wrapper').remove();
            }

            if (response.error) {
                addMessageToChat('ai', `Error generating code: ${response.error}`);
            } else {
                addCodeBlockToChat(response.code);
            }
        } catch (error) {
            console.error("Chat error:", error);
            const thinkingBubble = document.querySelector('.message-content .thinking');
            if (thinkingBubble) {
                thinkingBubble.closest('.message-wrapper').remove();
            }
            addMessageToChat('ai', `<span class="error-message">Error: ${error.message}</span>`);
        } finally {
            chatInputField.disabled = false;
            sendChatBtn.disabled = false;
            chatInputField.focus();
            saveChatState();
        }
    };

    sendChatBtn.addEventListener('click', sendChatMessage);
    
    chatInputField.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendChatMessage();
        }
    });

    // --- AI Model Configuration Logic ---
    saveModelSettingsBtn.addEventListener('click', async () => {
        const data = {
            ollama_base_url: baseUrlInput.value.trim() || undefined,
            embedding_model_name: embeddingModelInput.value.trim() || undefined,
            generative_model_name: generativeModelInput.value.trim() || undefined
        };
        const response = await callBackendApi('/api/ollama/config', 'POST', data);
        if (response.status === 'success') {
            alert(response.message);
            updateOllamaConfigUI();
            embeddingModelInput.value = '';
            generativeModelInput.value = '';
        } else {
            alert(`Failed to save: ${response.error}`);
        }
    });

    activateModelsBtn.addEventListener('click', async () => {
        const data = {
            active_embedding_model: activateEmbeddingSelect.value || undefined,
            active_generative_model: activateGenerativeSelect.value || undefined
        };
        const response = await callBackendApi('/api/ollama/active_models', 'POST', data);
        if (response.status === 'success') {
            alert(response.message);
            if (response.active_embedding_model) activeEmbeddingModelSpan.textContent = response.active_embedding_model;
            if (response.active_generative_model) activeGenerativeModelSpan.textContent = response.active_generative_model;
        } else {
            alert(`Failed to activate: ${response.error}`);
        }
    });

    // --- Initial Setup and View Management ---
    Object.values(rightPanelViews).forEach(view => {
        if (view) view.style.display = 'none';
    });

    // --- Application Initialization ---
    // Since there's no login, we immediately show the main app interface.
    const loadingSpinner = document.getElementById('loading-spinner');
    const loginPanel = document.getElementById('login-panel');
    
    if (appPanel) appPanel.style.display = 'flex';
    if (loginPanel) loginPanel.style.display = 'none';
    if (loadingSpinner) loadingSpinner.style.display = 'none';

    // Initial data load
    loadChatState();
    updateDocumentList();
});