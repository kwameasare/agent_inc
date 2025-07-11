import type { Task } from '../types';
import { PhaseCard } from './PhaseCard';
import { Bot } from 'lucide-react';

interface MissionControlProps {
  task: Task | null;
  onApprovePhase: (taskId: string, phaseId: string, approved: boolean, feedback?: string) => void;
}

export function MissionControl({ task, onApprovePhase }: MissionControlProps) {
  if (!task) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <Bot className="w-16 h-16 mx-auto text-gray-300 mb-4" />
          <h2 className="text-2xl font-semibold text-gray-600">Select a mission to begin</h2>
          <p className="text-gray-500">Your mission details will appear here.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      <div>
        <p className="text-sm text-gray-500">Mission Details</p>
        <h1 className="text-3xl font-bold tracking-tight">{task.description}</h1>
        <p className="text-sm text-gray-500">ID: {task.orchestratorId}</p>
      </div>
      <div className="space-y-4">
        {task.phases && task.phases.map((phase, index) => (
          <PhaseCard
            key={phase.id}
            phase={phase}
            phaseNumber={index + 1}
            onApprove={(approved, feedback) => onApprovePhase(task.orchestratorId!, phase.id, approved, feedback)}
          />
        ))}
      </div>
       {task.status === 'completed' && (
        <div className="p-6 bg-white border rounded-lg">
            <h3 className="font-semibold mb-2 text-green-600">Mission Complete: Final Report</h3>
            <pre className="text-sm bg-gray-100 p-4 rounded custom-scrollbar overflow-x-auto">{task.result}</pre>
        </div>
       )}
       {task.status === 'failed' && (
        <div className="p-6 bg-red-50 border border-red-200 rounded-lg">
            <h3 className="font-semibold mb-2 text-red-600">Mission Failed</h3>
            <p className="text-sm text-red-700">{task.error}</p>
        </div>
       )}
    </div>
  );
}
