// ui/src/components/ExpertDetails.tsx

import { useState } from 'react';
import { CheckCircle, XCircle, Clock, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import type { DomainExpert } from '../App';

// Helper to get the right icon based on status
const getExpertStatusIcon = (status: string) => {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-5 h-5 text-green-600" />;
    case 'running':
      return <Loader2 className="w-5 h-5 text-blue-600 animate-spin" />;
    case 'failed':
      return <XCircle className="w-5 h-5 text-red-600" />;
    default:
      return <Clock className="w-5 h-5 text-gray-500" />;
  }
};

interface ExpertDetailsProps {
  expert: DomainExpert;
}

export function ExpertDetails({ expert }: ExpertDetailsProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  // An expert's result can only be viewed if they have completed their task
  const canViewResult = expert.status === 'completed' && expert.result;

  return (
    <div className="activity-item activity-info p-4 my-2">
      <div
        className={`flex items-center justify-between ${canViewResult ? 'cursor-pointer' : ''}`}
        onClick={() => canViewResult && setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center space-x-3">
          {getExpertStatusIcon(expert.status)}
          <div>
            <p className="font-semibold text-gray-800">{expert.role}</p>
            <p className="text-sm text-gray-600">{expert.expertise}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
            <div className={`px-2 py-1 rounded text-xs font-bold ${
                expert.status === 'completed' ? 'bg-green-100 text-green-800' :
                expert.status === 'running' ? 'bg-blue-100 text-blue-800' :
                expert.status === 'failed' ? 'bg-red-100 text-red-800' :
                'bg-gray-100 text-gray-600'
            }`}>
                {expert.status.toUpperCase()}
            </div>
            {canViewResult && (
                isExpanded ? <ChevronUp className="w-5 h-5 text-gray-600" /> : <ChevronDown className="w-5 h-5 text-gray-600" />
            )}
        </div>
      </div>

      {/* This section will only render if the expert is expanded and has a result */}
      {isExpanded && canViewResult && (
        <div className="mt-4 pt-3 border-t border-gray-200">
          <h6 className="font-bold text-gray-800 text-sm mb-2">Expert's Report:</h6>
          <pre className="text-sm text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 p-3 rounded-lg border border-gray-200">
            {expert.result}
          </pre>
        </div>
      )}
    </div>
  );
}
