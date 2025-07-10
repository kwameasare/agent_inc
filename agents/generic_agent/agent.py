import os
import grpc
from concurrent import futures
import agent_pb2
import agent_pb2_grpc
from litellm import completion
import json
import logging
import traceback
import sys
import time
import pydantic
from typing import List

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Define Pydantic models for validation
class SubTask(pydantic.BaseModel):
    requested_persona: str
    task_details: str

class LLMDecision(pydantic.BaseModel):
    decision: str
    reason: str
    sub_tasks: List[SubTask] = []

class GenericAgentServicer(agent_pb2_grpc.GenericAgentServicer):
    """The implementation of a generic, multi-purpose agent."""

    def get_validated_decision(self, analysis_prompt: str) -> LLMDecision:
        """Get a validated decision from the LLM with retry logic."""
        for i in range(3):  # Retry loop
            try:
                response = completion(
                    model="gpt-4o", 
                    messages=[{"role": "user", "content": analysis_prompt}], 
                    response_format={"type": "json_object"},
                    timeout=60
                )
                data = json.loads(response.choices[0].message.content)
                validated_data = LLMDecision.model_validate(data)
                return validated_data
            except (json.JSONDecodeError, pydantic.ValidationError) as e:
                logger.warning(f"Validation failed (attempt {i+1}): {e}. Re-prompting with correction.")
                analysis_prompt += f"\n\nYour last response failed validation with this error: {e}. Please correct your response to match the required JSON schema."
        raise ValueError("Failed to get valid decision from LLM after 3 attempts.")

    def ExecuteTask(self, request, context):
        start_time = time.time()
        logger.info(f"🚀 Agent starting task {request.task_id} with persona: {request.persona_prompt[:100]}...")
        logger.info(f"🔍 Request details - Task ID: {request.task_id}, Instructions length: {len(request.task_instructions)}, Context data keys: {len(request.context_data)}")
        
        try:
            # Validate API key first
            if not os.getenv("OPENAI_API_KEY"):
                error_msg = "OPENAI_API_KEY environment variable not set"
                logger.error(f"❌ Task {request.task_id}: {error_msg}")
                return agent_pb2.TaskResult(
                    task_id=request.task_id, 
                    success=False, 
                    error_message=error_msg
                )

            # Special case: if this is a JSON response generator, skip analysis and go directly to execution
            if "JSON response generator" in request.persona_prompt or "ONLY output valid JSON" in request.persona_prompt:
                logger.info(f"📋 Task {request.task_id}: Detected JSON response request, executing directly")
                
                # Create a focused prompt for JSON generation
                json_prompt = f"""
{request.persona_prompt}

Task: {request.task_instructions}

Requirements:
- Output ONLY valid JSON
- No explanations or additional text
- Follow the exact format requested
- Ensure all JSON is properly formatted and complete
"""
                
                try:
                    json_response = completion(
                        model="gpt-4o",
                        messages=[{"role": "user", "content": json_prompt}],
                        response_format={"type": "json_object"},
                        timeout=120
                    )
                    
                    content = json_response.choices[0].message.content.strip()
                    logger.info(f"✅ Task {request.task_id}: Generated JSON response ({len(content)} chars)")
                    
                    return agent_pb2.TaskResult(
                        task_id=request.task_id,
                        success=True,
                        final_content=content
                    )
                    
                except Exception as e:
                    error_msg = f"JSON generation failed: {str(e)}"
                    logger.error(f"❌ Task {request.task_id}: {error_msg}")
                    return agent_pb2.TaskResult(
                        task_id=request.task_id,
                        success=False,
                        error_message=error_msg
                    )

            # This is the crucial "meta-prompt" or "Chain of Thought" prompt.
            # It instructs the LLM to first analyze the task's complexity.
            analysis_prompt = f"""
You are a project decomposition expert. Your first job is to analyze the following task and decide if it can be completed by a single specialist in one step, or if it requires a team of sub-specialists.

**The Task:**
---
{request.task_instructions}
---

**Current Context Data:**
{dict(request.context_data) if request.context_data else "No context provided"}

**Decision criteria:**
1.  **Is it simple?** Can a single AI agent with the persona "{request.persona_prompt}" reasonably produce a complete, high-quality response in a single pass? (e.g., "Write a Python function to sort a list", "Draft a user story for a login page").
2.  **Is it complex?** Does the task involve multiple distinct domains (e.g., frontend AND backend, hardware AND software), require multiple steps (e.g., design THEN code THEN test), or is it too broad (e.g., "Build a social media app")?

**Your Output Format:**
Respond with a JSON object.
- If the task is **simple**, respond with: {{"decision": "execute", "reason": "Your brief reason here."}}
- If the task is **complex**, respond with: {{"decision": "delegate", "reason": "Your brief reason here.", "sub_tasks": [{{"requested_persona": "Persona for sub-agent 1...", "task_details": "Specific task for sub-agent 1..."}}, ...]}}

Important: If you're synthesizing results from sub-agents (context data is provided), always choose "execute" to create the final synthesis.
"""

            logger.info(f"📊 Task {request.task_id}: Analyzing task complexity...")
            
            # Step 1: Call the LLM to make the execute/delegate decision with validation.
            try:
                decision_data = self.get_validated_decision(analysis_prompt)
                decision = decision_data.decision
                reason = decision_data.reason
                logger.info(f"✅ Task {request.task_id}: Received validated decision from LLM")
                
            except ValueError as e:
                # Return a gRPC error
                error_msg = str(e)
                logger.error(f"❌ Task {request.task_id}: {error_msg}")
                return agent_pb2.TaskResult(
                    task_id=request.task_id, 
                    success=False, 
                    error_message=error_msg
                )
            except Exception as e:
                error_msg = f"Failed to get decision from LLM: {str(e)}"
                logger.error(f"❌ Task {request.task_id}: {error_msg}")
                return agent_pb2.TaskResult(
                    task_id=request.task_id, 
                    success=False, 
                    error_message=error_msg
                )
            
            logger.info(f"🎯 Task {request.task_id}: Decision = {decision}, Reason = {reason}")

            # Step 2: Act on the decision.
            if decision == "delegate":
                # --- MODIFICATION: Enforce the rule ---
                if not request.can_delegate:
                    logger.warning(f"Task {request.task_id}: Agent chose 'delegate' but was not permitted. Overriding to 'execute'.")
                    decision = "execute"
                else:
                    logger.info(f"🔀 Task {request.task_id}: Delegating to sub-agents...")
                    # The agent has decided to break down the task.
                    # Populate the sub_tasks field for the orchestrator.
                    result = agent_pb2.TaskResult(task_id=request.task_id, success=True)
                    sub_tasks = decision_data.sub_tasks
                    
                    if not sub_tasks:
                        error_msg = "Decision was 'delegate' but no sub_tasks provided"
                        logger.error(f"❌ Task {request.task_id}: {error_msg}")
                        return agent_pb2.TaskResult(
                            task_id=request.task_id, 
                            success=False, 
                            error_message=error_msg
                        )
                    
                    for i, sub_task_data in enumerate(sub_tasks):
                        sub_task = result.sub_tasks.add()
                        sub_task.requested_persona = sub_task_data.requested_persona
                        sub_task.task_details = sub_task_data.task_details
                        logger.info(f"📋 Task {request.task_id}: Created sub-task {i+1}: {sub_task_data.requested_persona[:50]}...")
                    
                    elapsed = time.time() - start_time
                    logger.info(f"✅ Task {request.task_id}: Completed delegation in {elapsed:.2f}s with {len(sub_tasks)} sub-tasks")
                    return result

            if decision == "execute":  # Note: now an `if` instead of `elif`
                logger.info(f"⚡ Task {request.task_id}: Executing task directly...")
                # The task is simple enough to execute directly.
                # Use the provided persona to generate the final content.
                
                # Prepare the execution prompt with context if available
                execution_messages = [
                    {"role": "system", "content": request.persona_prompt}
                ]
                
                user_content = request.task_instructions
                if request.context_data:
                    logger.info(f"📝 Task {request.task_id}: Using context from {len(request.context_data)} sub-agents")
                    context_str = "\n\n**Context from sub-agents:**\n"
                    for persona, result in request.context_data.items():
                        context_str += f"\n**{persona}:**\n{result}\n"
                    user_content = context_str + "\n\n**Your Task:**\n" + user_content
                
                execution_messages.append({"role": "user", "content": user_content})
                
                try:
                    logger.info(f"🤔 Task {request.task_id}: Sending execution request to LLM...")
                    execution_response = completion(
                        model="gpt-4-turbo",  # Can be a different, potentially cheaper model
                        messages=execution_messages,
                        timeout=120  # 2 minute timeout for execution
                    )
                    logger.info(f"💬 Task {request.task_id}: Received response from LLM")
                    
                except Exception as e:
                    error_msg = f"Failed to get execution response from LLM: {str(e)}"
                    logger.error(f"❌ Task {request.task_id}: {error_msg}")
                    return agent_pb2.TaskResult(
                        task_id=request.task_id, 
                        success=False, 
                        error_message=error_msg
                    )
                
                final_content = execution_response.choices[0].message.content
                elapsed = time.time() - start_time
                content_length = len(final_content)
                
                logger.info(f"✅ Task {request.task_id}: Completed execution in {elapsed:.2f}s, generated {content_length} characters")
                
                return agent_pb2.TaskResult(
                    task_id=request.task_id,
                    final_content=final_content,
                    success=True
                )
            else:
                error_msg = f"Invalid decision from LLM: '{decision}'. Expected 'execute' or 'delegate'"
                logger.error(f"❌ Task {request.task_id}: {error_msg}")
                return agent_pb2.TaskResult(
                    task_id=request.task_id, 
                    success=False, 
                    error_message=error_msg
                )

        except Exception as e:
            # Capture full traceback for debugging
            error_traceback = traceback.format_exc()
            error_msg = f"Unexpected error in agent: {str(e)}"
            logger.error(f"💥 Task {request.task_id}: {error_msg}")
            logger.error(f"📊 Task {request.task_id}: Full traceback:\n{error_traceback}")
            
            return agent_pb2.TaskResult(
                task_id=request.task_id, 
                success=False, 
                error_message=f"{error_msg}\n\nTraceback:\n{error_traceback}"
            )

def serve(port):
    """Starts the gRPC server for the agent on a given port."""
    try:
        logger.info(f"🔧 Initializing Generic Agent server on port {port}...")
        
        # Validate environment
        if not os.getenv("OPENAI_API_KEY"):
            logger.error("❌ OPENAI_API_KEY environment variable not set!")
            sys.exit(1)
        else:
            logger.info("✅ OPENAI_API_KEY found")
        
        # Create and configure server with explicit options for better Go client compatibility
        options = [
            ('grpc.keepalive_time_ms', 10000),
            ('grpc.keepalive_timeout_ms', 3000),
            ('grpc.keepalive_permit_without_calls', True),
            ('grpc.http2.max_pings_without_data', 0),
            ('grpc.http2.min_time_between_pings_ms', 10000),
            ('grpc.http2.min_ping_interval_without_data_ms', 5000),
        ]
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10), options=options)
        agent_pb2_grpc.add_GenericAgentServicer_to_server(GenericAgentServicer(), server)
        
        # Bind to port
        listen_addr = f"0.0.0.0:{port}"
        server.add_insecure_port(listen_addr)
        
        # Start server
        server.start()
        logger.info(f"🚀 Generic Agent server successfully started on {listen_addr}")
        logger.info(f"🎯 Ready to receive tasks...")
        
        try:
            server.wait_for_termination()
        except KeyboardInterrupt:
            logger.info("🛑 Server interrupted by user")
        except Exception as e:
            logger.error(f"💥 Server error: {e}")
            
    except Exception as e:
        logger.error(f"💥 Failed to start server: {e}")
        logger.error(f"📊 Full traceback:\n{traceback.format_exc()}")
        sys.exit(1)

if __name__ == "__main__":
    # The port is passed as a command-line argument, which is essential for dynamic spawning.
    if len(sys.argv) > 1:
        port = sys.argv[1]
        logger.info(f"🔌 Using port from command line: {port}")
    else:
        port = "50051"  # Default port
        logger.info(f"🔌 Using default port: {port}")
    
    logger.info(f"🌟 Starting Generic Agent on port {port}...")
    serve(port)
