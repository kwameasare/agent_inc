import { Bot, Plus } from 'lucide-react';
import type { Task } from '../types';
import { Button } from './ui/Button';

interface MissionSidebarProps {
  tasks: Task[];
  activeTask: Task | null;
  setActiveTask: (task: Task) => void;
  onNewMission: () => void;
}

export function MissionSidebar({ tasks, activeTask, setActiveTask, onNewMission }: MissionSidebarProps) {
  return (
    <aside className="w-80 border-r border-gray-200 flex flex-col h-full bg-white">
      <div className="p-4 border-b border-gray-200 h-16 flex items-center">
        <div className="flex items-center space-x-3">
          <Bot className="w-8 h-8 text-blue-600" />
          <h2 className="text-xl font-bold">Agentic Control</h2>
        </div>
      </div>
      <div className="p-2 flex-1 overflow-y-auto custom-scrollbar">
        <div className="p-2">
            <Button onClick={onNewMission} className="w-full">
                <Plus className="w-4 h-4 mr-2" />
                New Mission
            </Button>
        </div>
        <nav className="space-y-1 p-2">
          {tasks.length === 0 ? (
            <div className="p-4 text-center text-gray-500">
              <div className="text-sm">No missions yet</div>
              <div className="text-xs text-gray-400 mt-1">
                Create your first mission to get started
              </div>
            </div>
          ) : (
            tasks.map(task => (
              <a
                key={task.id}
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  setActiveTask(task);
                }}
                className={`flex items-center justify-between p-2 rounded-md text-sm font-medium transition-colors ${
                  activeTask?.id === task.id
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                }`}
              >
                <span className="truncate pr-2">{task.description}</span>
                <span className={`capitalize px-2 py-0.5 text-xs rounded-full font-semibold ${
                  activeTask?.id === task.id ? 'bg-blue-500 text-white' : 'bg-gray-200 text-gray-600'
                }`}>
                  {task.status}
                </span>
              </a>
            ))
          )}
        </nav>
      </div>
    </aside>
  );
}
