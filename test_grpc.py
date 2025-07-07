#!/usr/bin/env python3

import grpc
import sys
import os

# Add the current directory to the Python path to import the generated files
sys.path.append('/Users/praise/dev/agent_inc/agents/generic_agent')

import agent_pb2
import agent_pb2_grpc

def test_grpc_connection():
    try:
        # Connect to the agent - test with orchestrator container on port 50060
        channel = grpc.insecure_channel('127.0.0.1:50060')
        
        # Wait for the channel to be ready
        grpc.channel_ready_future(channel).result(timeout=5)
        print("✅ Channel connected successfully")
        
        # Create client
        client = agent_pb2_grpc.GenericAgentStub(channel)
        
        # Create a simple test request
        request = agent_pb2.TaskRequest(
            task_id="test-123",
            persona_prompt="You are a helpful assistant.",
            task_instructions="Say hello and confirm you can receive this message.",
            context_data={}
        )
        
        print("📤 Sending test request...")
        
        # Make the call with a timeout
        response = client.ExecuteTask(request, timeout=30)
        
        print(f"✅ Response received:")
        print(f"   Success: {response.success}")
        print(f"   Final Content: {response.final_content}")
        print(f"   Error Message: {response.error_message}")
        print(f"   Sub-tasks: {len(response.sub_tasks)}")
        
        channel.close()
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    print("🧪 Testing gRPC connection to agent...")
    success = test_grpc_connection()
    if success:
        print("🎉 Test completed successfully!")
        sys.exit(0)
    else:
        print("💥 Test failed!")
        sys.exit(1)
