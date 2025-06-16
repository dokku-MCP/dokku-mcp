#!/usr/bin/env python3
"""
Simple test script to validate the MCP server is working correctly.
"""
import json
import subprocess
import sys
import time
import threading
import os
import select

def send_mcp_message(proc, message):
    """Send an MCP message to the server."""
    msg_json = json.dumps(message)
    print(f"→ Sending: {msg_json}")
    proc.stdin.write(msg_json + '\n')
    proc.stdin.flush()

def read_mcp_response(proc, timeout=5):
    """Read responses from the server with timeout."""
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            # Check if there's data available to read
            ready, _, _ = select.select([proc.stdout], [], [], 0.1)
            if ready:
                line = proc.stdout.readline()
                if not line:
                    break
                line = line.strip()
                if line:
                    print(f"← Received: {line}")
                    try:
                        response = json.loads(line)
                        return response
                    except json.JSONDecodeError:
                        print(f"← Non-JSON output: {line}")
            
            # Check for stderr output
            ready, _, _ = select.select([proc.stderr], [], [], 0.1)
            if ready:
                stderr_line = proc.stderr.readline()
                if stderr_line:
                    print(f"← Stderr: {stderr_line.strip()}")
                    
        except Exception as e:
            print(f"Error reading response: {e}")
            break
    return None

def test_mcp_server():
    """Test the MCP server."""
    print("Starting MCP server test...")
    
    # Set environment variables for debug logging
    env = os.environ.copy()
    env['DOKKU_MCP_LOG_LEVEL'] = 'debug'
    env['DOKKU_MCP_LOG_FORMAT'] = 'text'
    
    # Start the server
    proc = None
    try:
        proc = subprocess.Popen(
            ['./build/dokku-mcp'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=env
        )
        
        print("MCP server started, waiting for initialization...")
        
        # Give server time to initialize and capture any immediate output
        time.sleep(3)
        
        # Check if process is still running
        if proc.poll() is not None:
            print(f"❌ Server exited early with code: {proc.poll()}")
            stderr_output = proc.stderr.read()
            stdout_output = proc.stdout.read()
            if stderr_output:
                print(f"Server stderr: {stderr_output}")
            if stdout_output:
                print(f"Server stdout: {stdout_output}")
            return
        
        print("Server is running, checking for any immediate output...")
        
        # Check for any immediate output
        ready, _, _ = select.select([proc.stderr], [], [], 0.1)
        if ready:
            stderr_line = proc.stderr.read()
            if stderr_line:
                print(f"Server startup logs: {stderr_line}")
        
        # Send initialization message
        init_message = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {}
                },
                "clientInfo": {
                    "name": "test-client",
                    "version": "1.0.0"
                }
            }
        }
        
        send_mcp_message(proc, init_message)
        
        # Wait for response
        print("Waiting for initialization response...")
        response = read_mcp_response(proc, timeout=10)
        
        if response:
            print("✅ Server responded to initialization!")
            print(f"Response: {json.dumps(response, indent=2)}")
            
            # Send a tools/list request
            tools_message = {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/list"
            }
            
            send_mcp_message(proc, tools_message)
            tools_response = read_mcp_response(proc, timeout=5)
            
            if tools_response:
                print("✅ Server responded to tools/list!")
                print(f"Tools response: {json.dumps(tools_response, indent=2)}")
            else:
                print("❌ No response to tools/list")
        else:
            print("❌ No response to initialization")
            
            # Check for any remaining stderr
            try:
                proc.stderr.settimeout(1)
                stderr_output = proc.stderr.read()
                if stderr_output:
                    print(f"Server stderr: {stderr_output}")
            except:
                pass
        
    except Exception as e:
        print(f"Error testing MCP server: {e}")
    finally:
        # Cleanup
        if proc:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()

if __name__ == "__main__":
    test_mcp_server() 