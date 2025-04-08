import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Divider, List, ListItem, ListItemText, Chip, Button, Tabs, Tab } from '@mui/material';
import { WebhookNotification } from '../services/api';
import { getWebhooks, clearWebhooks, getWebhooksFromServer, getWebhookServerUrl } from '../services/webhookHandler';

interface WebhookReceiverProps {
  refreshInterval?: number;
}

const WebhookReceiver: React.FC<WebhookReceiverProps> = ({ refreshInterval = 2000 }) => {
  const [webhooks, setWebhooks] = useState<WebhookNotification[]>([]);
  const [isListening, setIsListening] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tabValue, setTabValue] = useState(0);
  const [serverConnected, setServerConnected] = useState(false);

  // Function to refresh webhooks from localStorage
  const refreshLocalWebhooks = () => {
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

  // Function to refresh webhooks from server
  const refreshServerWebhooks = async () => {
    try {
      const serverWebhooks = await getWebhooksFromServer('twitter-solver');
      setWebhooks(serverWebhooks);
      setServerConnected(true);
      setIsListening(true);
      setError(null);
    } catch (err) {
      setServerConnected(false);
      setError(err instanceof Error ? err.message : 'Failed to connect to webhook server');
      setIsListening(false);
    }
  };

  // Function to refresh webhooks based on selected tab
  const refreshWebhooks = () => {
    if (tabValue === 0) {
      refreshLocalWebhooks();
    } else {
      refreshServerWebhooks();
    }
  };

  useEffect(() => {
    // Initial load of webhooks
    refreshWebhooks();

    // Set up polling interval to check for new webhooks
    const interval = setInterval(refreshWebhooks, refreshInterval);

    // Clean up interval on unmount
    return () => clearInterval(interval);
  }, [refreshInterval, tabValue]);

  const handleClearWebhooks = () => {
    if (tabValue === 0) {
      clearWebhooks();
      setWebhooks([]);
    } else {
      // Server webhooks can't be cleared from the client
      setError('Server webhooks cannot be cleared from the client');
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
            label={isListening ? "Listening for webhooks..." : "Not listening"} 
            color={isListening ? "success" : "default"} 
            size="small" 
            sx={{ mr: 1 }}
          />
          <Button 
            variant="outlined" 
            size="small" 
            onClick={handleClearWebhooks}
            disabled={webhooks.length === 0 || tabValue === 1}
          >
            Clear All
          </Button>
        </Box>
      </Box>

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
          {tabValue === 0 
            ? "No webhooks received yet. Use the mock webhook URL above in your transaction submission."
            : "No webhooks received yet. Make sure the webhook server is running and clients are sending webhooks."}
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
                    {webhook.client && (
                      <Chip 
                        label={webhook.client} 
                        size="small" 
                        variant="outlined"
                      />
                    )}
                  </Box>
                }
                secondary={
                  <>
                    <Typography variant="body2">
                      Chain: {webhook.chain} | From: {webhook.fromAddress.substring(0, 10)}...
                    </Typography>
                    <Typography variant="body2">
                      Amount: {typeof webhook.amount === 'string' ? webhook.amount : JSON.stringify(webhook.amount)} {webhook.fromToken} | 
                      Time: {new Date(parseInt(typeof webhook.timestamp === 'string' ? webhook.timestamp : String(webhook.timestamp)) * 1000).toLocaleString()}
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
