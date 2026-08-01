# Chatbot Project

This project sets up a simple chatbot using Flask that simulates a technical interview guide based on the Agent Harness document.

## Setup Instructions

1. **Install dependencies:**
   ```bash
   pip install flask
   ```

2. **Run the server:**
   ```bash
   flask run
   ```

3. **Navigate to the server URL:** Open [http://127.0.0.1:5000](http://127.0.0.1:5000) in your browser.

## Project Structure

- `app.py`: Main application file.
- `templates/index.html`: HTML template for chatbot interface.
- `static/`: Folder for static files.
- `docs/`: Location to maintain project documentation.

## Main Features

- **Interactive Interview Guide:** Simulates questions from the Agent Harness document.
- **Basic Prompt Response Handling:** Captures and validates user input.

## Future Enhancements

- **Expand question variety**
- **Add user analytics**
- **Integrate with a database for persistent user data**
