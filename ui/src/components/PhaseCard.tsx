import { CheckCircle, XCircle, Clock, Loader2, ChevronDown, ChevronUp, Users, FileText } from 'lucide-react';
import type { ProjectPhase, DomainExpert } from '../types';
import { useState } from 'react';
import { Button } from './ui/Button';

interface PhaseCardProps {
  phase: ProjectPhase;
  phaseNumber: number;
  onApprove: (approved: boolean, feedback?: string) => void;
}

const getStatusIcon = (status: string) => {
    switch(status) {
        case 'completed':
        case 'approved':
            return <CheckCircle className="w-5 h-5 text-green-500" />;
        case 'running':
            return <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />;
        case 'awaiting_approval':
            return <Clock className="w-5 h-5 text-yellow-500" />;
        case 'rejected':
            return <XCircle className="w-5 h-5 text-red-500" />;
        default:
            return <Clock className="w-5 h-5 text-gray-400" />;
    }
};

export function PhaseCard({ phase, phaseNumber, onApprove }: PhaseCardProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [feedback, setFeedback] = useState('');

  return (
    <div className="bg-white border rounded-lg overflow-hidden transition-all">
      <div
        className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center space-x-4">
          <div className="w-10 h-10 flex items-center justify-center bg-gray-100 rounded-full">
            {getStatusIcon(phase.status)}
          </div>
          <div>
            <h3 className="font-semibold">Phase {phaseNumber}: {phase.name}</h3>
            <p className="text-sm text-gray-600">{phase.description}</p>
          </div>
        </div>
        {isExpanded ? <ChevronUp className="w-5 h-5 text-gray-400" /> : <ChevronDown className="w-5 h-5 text-gray-400" />}
      </div>

      {isExpanded && (
        <div className="p-4 border-t border-gray-200 bg-gray-50">
          {phase.experts && phase.experts.length > 0 && (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <Users className="w-4 h-4 text-gray-600" />
                <h4 className="font-medium text-gray-900">Domain Experts ({phase.experts.length})</h4>
              </div>
              <div className="grid gap-3">
                {phase.experts.map((expert, index) => (
                  <ExpertResult key={index} expert={expert} />
                ))}
              </div>
            </div>
          )}
          
          {phase.status === 'awaiting_approval' && (
            <div className="mt-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
              <h5 className="font-medium text-yellow-800 mb-3">Approval Required</h5>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-yellow-800 mb-1">
                    Feedback (optional)
                  </label>
                  <textarea
                    className="w-full px-3 py-2 border border-yellow-300 rounded-md text-sm bg-white focus:ring-2 focus:ring-yellow-500 focus:border-transparent"
                    rows={3}
                    placeholder="Provide feedback or instructions..."
                    value={feedback}
                    onChange={(e) => setFeedback(e.target.value)}
                  />
                </div>
                
                <div className="flex space-x-3">
                  <Button
                    onClick={() => onApprove(true, feedback)}
                    className="flex items-center"
                  >
                    <CheckCircle className="w-4 h-4 mr-2" />
                    Approve
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={() => onApprove(false, feedback)}
                    className="flex items-center"
                  >
                    <XCircle className="w-4 h-4 mr-2" />
                    Reject
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

const ExpertResult = ({ expert }: { expert: DomainExpert }) => {
    const [showResult, setShowResult] = useState(false);
    
    return (
        <div className="bg-white border rounded-lg p-3">
            <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                    {getStatusIcon(expert.status)}
                    <div>
                        <div className="font-medium text-sm">{expert.role}</div>
                        <div className="text-xs text-gray-500">{expert.expertise}</div>
                    </div>
                </div>
                {expert.result && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowResult(!showResult)}
                        className="text-xs"
                    >
                        <FileText className="w-3 h-3 mr-1" />
                        {showResult ? 'Hide' : 'View'} Result
                    </Button>
                )}
            </div>
            {showResult && expert.result && (
                <div className="mt-3 p-3 bg-gray-100 rounded text-sm font-mono max-h-40 overflow-y-auto custom-scrollbar">
                    {expert.result}
                </div>
            )}
        </div>
    );
}
