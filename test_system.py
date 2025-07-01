#!/usr/bin/env python3

import grpc
import sys
import os

# Add the current directory to the Python path so we can import the protobuf files
sys.path.append('/Users/praise/dev/agent_inc/agents/generic_agent')

import agent_pb2
import agent_pb2_grpc

def test_agent_connection():
    """Test if we can connect to an agent container"""
    print("Testing agent container connection...")
    
    # Start a test container
    import subprocess
    import time
    
    print("Starting test container...")
    result = subprocess.run([
        'docker', 'run', '--rm', '-d', '-p', '50061:50061',
        '-e', f'OPENAI_API_KEY={os.getenv("OPENAI_API_KEY")}',
        'agentic-engineering-system_generic_agent', 
        'python', 'agent.py', '50061'
    ], capture_output=True, text=True)
    
    if result.returncode != 0:
        print(f"Failed to start container: {result.stderr}")
        return False
    
    container_id = result.stdout.strip()
    print(f"Container started: {container_id}")
    
    # Wait for the server to start
    time.sleep(3)
    
    try:
        # Connect to the agent
        channel = grpc.insecure_channel('localhost:50061')
        stub = agent_pb2_grpc.GenericAgentStub(channel)
        
        # Create a simple test request
        request = agent_pb2.TaskRequest(
            task_id="test-123",
            persona_prompt="You are a helpful assistant.",
            task_instructions="Respond with exactly 'Hello World' and nothing else.",
            context_data={}
        )
        
        print("Sending test request...")
        response = stub.ExecuteTask(request, timeout=30)
        
        print(f"Response received:")
        print(f"  Success: {response.success}")
        print(f"  Final Content: {response.final_content[:100]}...")
        print(f"  Sub-tasks: {len(response.sub_tasks)}")
        
        if response.success:
            print("✅ Test PASSED! Agent container is working correctly.")
            success = True
        else:
            print(f"❌ Test FAILED: {response.error_message}")
            success = False
            
    except Exception as e:
        print(f"❌ Test FAILED with exception: {e}")
        success = False
    finally:
        # Clean up
        print("Stopping test container...")
        subprocess.run(['docker', 'stop', container_id], capture_output=True)
        
    return success

if __name__ == "__main__":
    if test_agent_connection():
        print("\n🎉 System is ready to run!")
        print("You can now run: cd orchestrator && go run .")
    else:
        print("\n💥 System test failed. Please check the logs above.")
        sys.exit(1)
