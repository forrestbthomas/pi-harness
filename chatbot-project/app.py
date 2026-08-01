from flask import Flask, render_template, request, jsonify

app = Flask(__name__)

@app.route('/', methods=['GET'])

def index():
    return render_template('index.html')

@app.route('/chat', methods=['POST'])

def chat():
    user_message = request.form['message']
    bot_response = generate_response(user_message)
    return jsonify({'response': bot_response})

def generate_response(user_message):
    # Simple response logic; expand as needed for real questions
    if 'project goals' in user_message.lower():
        return "The primary goals of the Agent Harness project include providing a reproducible environment for coding agents and evaluating Golang controllers."
    elif 'conventions' in user_message.lower():
        return "Conventions include keeping changes minimal, using Kubebuilder, and managing API keys securely."
    elif 'commands' in user_message.lower():
        return "Useful commands for the workflow include `make pi-eval`, `make pi`, and `make pi-config-check`."
    else:
        return "I'm here to help with project questions. Try asking about goals, conventions, or commands."

if __name__ == '__main__':
    app.run(debug=True)
