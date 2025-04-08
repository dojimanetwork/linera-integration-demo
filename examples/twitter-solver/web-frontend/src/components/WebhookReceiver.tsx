import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Divider, List, ListItem, ListItemText, Chip, Button, Tabs, Tab } from '@mui/material';
import { WebhookNotification } from '../services/api';
import { getWebhooks, clearWebhooks, getWebhooksFromServer, getWebhookServerUrl } from '../services/webhookHandler';

interface WebhookReceiverProps {
  refreshInterval?: number;
}

const WebhookReceiver: React.FC<WebhookReceiverProps> = ({ refreshInterval = 2000 }) => {
  const [webhooks, setWebhooks] = useState<WebhookNotification[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tabValue, setTabValue] = useState(0);
  const [serverConnected, setServerConnected] = useState(false);

  const fetchWebhooks = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const serverWebhooks = await getWebhooksFromServer();
      setWebhooks(serverWebhooks);
    } catch (err) {
      setError('Failed to fetch webhooks from server');
      console.error('Error fetching webhooks:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const clearWebhooks = async () => {
    try {
      await clearWebhooks();
      setWebhooks([]);
    } catch (err) {
      setError('Failed to clear webhooks');
      console.error('Error clearing webhooks:', err);
    }
  };

  useEffect(() => {
    fetchWebhooks();
  }, []);

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'success':
        return 'text-green-500';
      case 'error':
        return 'text-red-500';
      default:
        return 'text-gray-500';
    }
  };

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  return (
    <Paper elevation={3} sx={{ p: 3, mb: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">
          Webhook Receiver
        </Typography>
        <Box>
          <Chip 
            label={isLoading ? 'Loading...' : 'Refresh'} 
            color={isLoading ? "success" : "default"} 
            size="small" 
            sx={{ mr: 1 }}
          />
          <Button 
            variant="outlined" 
            size="small" 
            onClick={fetchWebhooks}
            disabled={isLoading}
          >
            Refresh
          </Button>
          <Button 
            variant="outlined" 
            size="small" 
            onClick={clearWebhooks}
            disabled={webhooks.length === 0}
          >
            Clear All
          </Button>
        </Box>
      </Box>

      {error && (
        <Typography color="error" variant="body2" sx={{ mb: 2 }}>
          Error: {error}
        </Typography>
      )}

      <Tabs value={tabValue} onChange={handleTabChange} sx={{ mb: 2 }}>
        <Tab label="Local Storage" />
        <Tab label="Webhook Server" />
      </Tabs>
      
      {tabValue === 0 && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Webhook URL for testing:
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="body1" sx={{ 
              fontFamily: 'monospace', 
              bgcolor: 'background.paper', 
              p: 1, 
              borderRadius: 1,
              flexGrow: 1,
              wordBreak: 'break-all'
            }}>
              mock-webhook://receive
            </Typography>
            <Chip 
              label="Copy" 
              onClick={() => navigator.clipboard.writeText('mock-webhook://receive')} 
              size="small" 
              color="primary" 
            />
          </Box>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            Note: This is a mock webhook URL. When selected, webhooks will be stored locally in your browser.
          </Typography>
        </Box>
      )}

      {tabValue === 1 && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Webhook Server URL:
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="body1" sx={{ 
              fontFamily: 'monospace', 
              bgcolor: 'background.paper', 
              p: 1, 
              borderRadius: 1,
              flexGrow: 1,
              wordBreak: 'break-all'
            }}>
              {getWebhookServerUrl()}
            </Typography>
            <Chip 
              label="Copy" 
              onClick={() => navigator.clipboard.writeText(getWebhookServerUrl())} 
              size="small" 
              color="primary" 
            />
          </Box>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            {serverConnected 
              ? "Connected to webhook server. Receiving webhooks from all clients." 
              : "Not connected to webhook server. Make sure the server is running."}
          </Typography>
        </Box>
      )}

      <Divider sx={{ my: 2 }} />

      <Typography variant="subtitle1" gutterBottom>
        Received Webhooks ({webhooks.length})
      </Typography>

      {webhooks.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {tabValue === 0 
            ? "No webhooks received yet. Use the mock webhook URL above in your transaction submission."
            : "No webhooks received yet. Make sure the webhook server is running and clients are sending webhooks."}
        </Typography>
      ) : (
        <div className="space-y-4">
          {webhooks.map((webhook, index) => (
            <div
              key={index}
              className="p-4 border rounded shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start">
                <div>
                  <p className={`font-semibold ${getStatusColor(webhook.status)}`}>
                    Status: {webhook.status}
                  </p>
                  <p>Transaction Hash: {webhook.txHash}</p>
                  <p>Chain: {webhook.chain}</p>
                  <p>From: {webhook.fromAddress}</p>
                  <p>
                    Amount: {typeof webhook.amount === 'string' ? webhook.amount : JSON.stringify(webhook.amount)} {webhook.fromToken}
                  </p>
                  <p>
                    Time: {new Date(parseInt(typeof webhook.timestamp === 'string' ? webhook.timestamp : String(webhook.timestamp)) * 1000).toLocaleString()}
                  </p>
                  {webhook.client && <p>Client: {webhook.client}</p>}
                </div>
                {webhook.data?.screenshot && (
                  <div className="ml-4">
                    <img
                      src={`http://localhost:3005/screenshots/${webhook.data.screenshot}`}
                      alt="Transaction Screenshot"
                      className="max-w-xs rounded shadow-sm"
                    />
                  </div>
                )}
              </div>
              {webhook.data?.error && (
                <div className="mt-2 p-2 bg-red-50 text-red-700 rounded">
                  <p className="font-semibold">Error Details:</p>
                  <p>{webhook.data.error}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </Paper>
  );
};

export default WebhookReceiver; 
