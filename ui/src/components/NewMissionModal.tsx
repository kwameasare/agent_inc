import { Send, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { Button } from './ui/Button';

interface NewMissionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (description: string) => void;
  isLoading: boolean;
}

export function NewMissionModal({ isOpen, onClose, onSubmit, isLoading }: NewMissionModalProps) {
  const [description, setDescription] = useState('');

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (description.trim() && !isLoading) {
      onSubmit(description.trim());
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-2xl" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-xl font-bold mb-4">Launch a New Mission</h2>
        <p className="text-sm text-gray-600 mb-4">Define a complex objective for the AI agent team. Be as descriptive as possible.</p>
        <form onSubmit={handleSubmit}>
          <textarea
            className="w-full p-2 border rounded-md text-sm h-40 bg-white focus:ring-2 focus:ring-blue-500"
            placeholder="Example: Design a complete architecture for a scalable, real-time chat application including database schema, backend APIs, and frontend components..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
          <div className="flex justify-end space-x-2 mt-4">
            <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
            <Button type="submit" disabled={isLoading || !description.trim()}>
              {isLoading ? <Loader2 className="w-4 h-4 animate-spin mr-2"/> : <Send className="w-4 h-4 mr-2" />}
              <span>{isLoading ? "Launching..." : "Launch Mission"}</span>
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
