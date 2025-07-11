import { useState, useEffect } from 'react';
import { MissionSidebar } from './components/MissionSidebar';
import { MissionControl } from './components/MissionControl';
import { NewMissionModal } from './components/NewMissionModal';
import { Button } from './components/ui/Button';
import { Plus } from 'lucide-react';
import type { Task } from './types';

export default function App() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  console.log('App rendering with tasks:', tasks, 'activeTask:', activeTask); // Debug log

  // Load tasks from localStorage on component mount
  useEffect(() => {
    const savedTasks = localStorage.getItem('agent_inc_tasks');
    const savedCurrentTaskId = localStorage.getItem('agent_inc_current_task');
    
    if (savedTasks) {
      try {
        const parsedTasks = JSON.parse(savedTasks).map((task: any) => ({
          ...task,
          timestamp: new Date(task.timestamp)
        }));
        setTasks(parsedTasks);
        
        if (savedCurrentTaskId) {
          const currentTask = parsedTasks.find((task: Task) => task.id === savedCurrentTaskId);
          if (currentTask) {
            setActiveTask(currentTask);
          }
        }
      } catch (error) {
        console.error('Failed to load tasks from localStorage:', error);
      }
    }
  }, []);

  // Save tasks to localStorage whenever tasks change
  useEffect(() => {
    localStorage.setItem('agent_inc_tasks', JSON.stringify(tasks));
  }, [tasks]);

  // Save current task ID to localStorage whenever it changes
  useEffect(() => {
    if (activeTask) {
      localStorage.setItem('agent_inc_current_task', activeTask.id);
    } else {
      localStorage.removeItem('agent_inc_current_task');
    }
  }, [activeTask]);

  // This effect manages the real-time connection to the backend for the active task.
  // It uses Server-Sent Events (SSE) for efficient, one-way data flow from server to client.
  // It only runs when the active task changes or its status becomes 'running'.
  useEffect(() => {
    if (activeTask?.status === 'running' || activeTask?.status === 'planning') {
      const eventSource = new EventSource(`http://localhost:8081/api/task/${activeTask.orchestratorId}/events`);
      
      eventSource.onmessage = (event) => {
        const updatedTaskData = JSON.parse(event.data);
        // We must parse the timestamp string back into a Date object.
        const taskToUpdate: Task = {
            ...updatedTaskData,
            timestamp: new Date(updatedTaskData.Started),
            orchestratorId: activeTask.orchestratorId || updatedTaskData.id,
            phases: updatedTaskData.phases || []
        };
        
        // Update both the active task view and the list in the sidebar.
        setActiveTask(taskToUpdate);
        setTasks(prev => prev.map(t => t.id === taskToUpdate.id ? taskToUpdate : t));

        // If the task is finished (completed or failed), we close the connection.
        if (taskToUpdate.status === 'completed' || taskToUpdate.status === 'failed' || taskToUpdate.status === 'error') {
            eventSource.close();
        }
      };

      eventSource.onerror = () => {
        // In a real application, you would implement more robust error handling here,
        // such as showing a "connection lost" message to the user.
        console.error("SSE connection error. Closing connection.");
        eventSource.close();
      };

      // The cleanup function is crucial. It ensures that when the component unmounts
      // or the active task changes, we close the old SSE connection to prevent memory leaks.
      return () => {
        eventSource.close();
      };
    }
  }, [activeTask?.id, activeTask?.status]);

  // WebSocket connection for real-time updates
  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8081/ws');
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'task_created' || data.type === 'task_updated') {
        const updatedTask: Task = {
          ...data.data,
          timestamp: new Date(data.data.started),
          orchestratorId: data.data.id,
          phases: data.data.phases || []
        };
        
        setTasks(prev => {
          const existingTaskIndex = prev.findIndex(t => t.orchestratorId === updatedTask.orchestratorId);
          if (existingTaskIndex >= 0) {
            const newTasks = [...prev];
            newTasks[existingTaskIndex] = updatedTask;
            return newTasks;
          } else {
            return [updatedTask, ...prev];
          }
        });
        
        // Update active task if it's the one being updated
        if (activeTask?.orchestratorId === updatedTask.orchestratorId) {
          setActiveTask(updatedTask);
        }
      }
    };

    return () => {
      ws.close();
    };
  }, [activeTask?.orchestratorId]);

  const submitNewMission = async (description: string) => {
    setIsLoading(true);
    setIsModalOpen(false);

    try {
      const response = await fetch('http://localhost:8081/api/task', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ task: description }),
      });
      if (!response.ok) throw new Error("Failed to submit task to orchestrator.");
      const submitResult = await response.json();
      
      const newTask: Task = {
        id: submitResult.id, // Use the canonical ID from the backend
        orchestratorId: submitResult.id,
        description,
        status: 'planning', // The initial status after submission
        timestamp: new Date(),
        phases: [], // Phases will be populated by the first SSE event
      };

      // Add the new task to the top of the list and set it as active.
      setTasks(prev => [newTask, ...prev]);
      setActiveTask(newTask);
    } catch (error) {
      console.error("Failed to submit new mission:", error);
      // TODO: Implement user-facing error notification (e.g., a toast message).
    } finally {
      setIsLoading(false);
    }
  };

  const handleApprovePhase = async (taskId: string, phaseId: string, approved: boolean, feedback?: string) => {
    // This function will be passed down to the MissionControl component.
    // It's responsible for sending the user's approval decision to the backend.
    try {
        await fetch('http://localhost:8081/api/phases/approve', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ taskId, phaseId, approved, userFeedback: feedback }),
        });
        // The backend will process this, and the result will trigger a new SSE event,
        // which will automatically update the UI. We don't need to manually set state here.
    } catch (error) {
        console.error("Failed to approve phase:", error);
    }
  };

  return (
    <div className="flex h-screen w-screen bg-gray-50 text-gray-900 font-sans">
      <MissionSidebar
        tasks={tasks}
        activeTask={activeTask}
        setActiveTask={setActiveTask}
        onNewMission={() => setIsModalOpen(true)}
      />

      <main className="flex-1 flex flex-col h-screen">
        <header className="flex items-center justify-between border-b border-gray-200 h-16 px-6 shrink-0 bg-white">
          <h1 className="text-lg font-semibold truncate">
            {activeTask ? `Mission: ${activeTask.description}` : "Mission Control"}
          </h1>
          <Button onClick={() => setIsModalOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            New Mission
          </Button>
        </header>
        
        <div className="flex-1 overflow-y-auto p-6 custom-scrollbar">
          <MissionControl task={activeTask} onApprovePhase={handleApprovePhase} />
        </div>
      </main>

      <NewMissionModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={submitNewMission}
        isLoading={isLoading}
      />
    </div>
  );
}
