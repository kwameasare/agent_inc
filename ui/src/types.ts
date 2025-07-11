export interface DomainExpert {
  role: string;
  expertise: string;
  persona: string;
  task: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  result?: string;
}

export interface ProjectPhase {
  id: string;
  name: string;
  description: string;
  status: 'pending' | 'approved' | 'running' | 'completed' | 'rejected' | 'awaiting_approval';
  experts: DomainExpert[];
  results?: { [key: string]: string };
  startTime?: string;
  endTime?: string;
  approved: boolean;
  userFeedback?: string;
}

export interface Task {
  id: string;
  description: string;
  status: 'pending' | 'planning' | 'running' | 'completed' | 'error' | 'failed';
  result?: string;
  error?: string;
  timestamp: Date;
  orchestratorId?: string;
  phases: ProjectPhase[];
}
