import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Divider, List, ListItem, ListItemText, Chip, Button } from '@mui/material';
import { WebhookNotification } from '../services/api';
import { getWebhooks, clearWebhooks } from '../services/webhookHandler';

interface WebhookReceiverProps {
  refreshInterval?: number;
}

const WebhookReceiver: React.FC<WebhookReceiverProps> = ({ refreshInterval = 2000 }) => {
  const [webhooks, setWebhooks] = useState<WebhookNotification[]>([]);
  const [isListening, setIsListening] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Function to refresh webhooks from localStorage
  const refreshWebhooks = () => {
    try {
      const storedWebhooks = getWebhooks();
      setWebhooks(storedWebhooks);
      setIsListening(true);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retrieve webhooks');
      setIsListening(false);
    }
  };

  useEffect(() => {
    // Initial load of webhooks
    refreshWebhooks();

    // Set up polling interval to check for new webhooks
    const interval = setInterval(refreshWebhooks, refreshInterval);

    // Clean up interval on unmount
    return () => clearInterval(interval);
  }, [refreshInterval]);

  const handleClearWebhooks = () => {
    clearWebhooks();
    setWebhooks([]);
  };

  return (
    <Paper elevation={3} sx={{ p: 3, mb: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">
          Webhook Receiver
        </Typography>
        <Box>
          <Chip 
            label={isListening ? "Listening for webhooks..." : "Not listening"} 
            color={isListening ? "success" : "default"} 
            size="small" 
            sx={{ mr: 1 }}
          />
          <Button 
            variant="outlined" 
            size="small" 
            onClick={handleClearWebhooks}
            disabled={webhooks.length === 0}
          >
            Clear All
          </Button>
        </Box>
      </Box>
      
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

      {error && (
        <Typography color="error" variant="body2" sx={{ mb: 2 }}>
          Error: {error}
        </Typography>
      )}

      <Divider sx={{ my: 2 }} />

      <Typography variant="subtitle1" gutterBottom>
        Received Webhooks ({webhooks.length})
      </Typography>

      {webhooks.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          No webhooks received yet. Use the mock webhook URL above in your transaction submission.
        </Typography>
      ) : (
        <List>
          {webhooks.map((webhook, index) => (
            <ListItem key={index} divider={index < webhooks.length - 1}>
              <ListItemText
                primary={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="subtitle2">
                      Transaction: {webhook.txHash.substring(0, 10)}...
                    </Typography>
                    <Chip 
                      label={webhook.status} 
                      color={webhook.status === 'success' ? 'success' : 'error'} 
                      size="small" 
                    />
                  </Box>
                }
                secondary={
                  <>
                    <Typography variant="body2">
                      Chain: {webhook.chain} | From: {webhook.fromAddress.substring(0, 10)}...
                    </Typography>
                    <Typography variant="body2">
                      Amount: {webhook.amount} {webhook.fromToken} | 
                      Time: {new Date(parseInt(webhook.timestamp) * 1000).toLocaleString()}
                    </Typography>
                  </>
                }
              />
            </ListItem>
          ))}
        </List>
      )}
    </Paper>
  );
};

export default WebhookReceiver; 