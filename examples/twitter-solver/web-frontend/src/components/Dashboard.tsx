import React, {useEffect, useState} from 'react';
import { Link, Box, Typography, Button, Paper, Avatar, TextField, FormControlLabel, Switch } from '@mui/material';
import { useAuth } from '../contexts/AuthContext';
import api, {Tweet, WebhookNotification} from "../services/api";
import CameraAltIcon from "@mui/icons-material/CameraAlt";
import {useNavigate} from 'react-router-dom';
import UserProfile from './UserProfile';
import WebhookReceiver from './WebhookReceiver';
import { addWebhook } from '../services/webhookHandler';

export const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [tweet, setTweet] = useState<Tweet | null>(null);
  const [screenshotUrl, setScreenshotUrl] = useState<string | null>(null);
  const [profileScreenshotUrl, setProfileScreenshotUrl] = useState<string | null>(null);
  const [isTakingScreenshot, setIsTakingScreenshot] = useState(false);
  const [isTakingProfileScreenshot, setIsTakingProfileScreenshot] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [txHash, setTxHash] = useState('');
  const [chain, setChain] = useState('solana');
  const [webhookUrl, setWebhookUrl] = useState('');
  const [useWebhook, setUseWebhook] = useState(false);
  const [webhookStatus, setWebhookStatus] = useState<WebhookNotification | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedWebhookOption, setSelectedWebhookOption] = useState('none');
  const navigate = useNavigate();

  // Predefined webhook URLs
  const webhookOptions = [
    { id: 'none', label: 'No Webhook', url: '' },
    { id: 'builtin', label: 'Built-in Webhook Receiver', url: 'http://localhost:3005/webhook' },
    { id: 'server', label: 'Webhook Server', url: 'http://localhost:3006/webhook' },
    { id: 'custom', label: 'Custom URL', url: '' }
  ];

  useEffect(() => {
    const handleLatestTweet = async () => {
      try {
        // Fetch the latest tweet
        const tweetResponse = await api.getLatestTweet();
        if (tweetResponse.status === 'success' && tweetResponse.tweet) {
          setTweet(tweetResponse.tweet);
        }
      }catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred');
      }
    }
    handleLatestTweet()
  },[])

  const handleTakeScreenshot = async () => {
    if (!tweet) return;

    setIsTakingScreenshot(true);
    try {
      const tweetUrl = `https://twitter.com/user/status/${tweet.id}`;
      const response = await api.takeScreenshot(tweetUrl);
      if (response.status === 'success') {
        setScreenshotUrl(response.screenshot_url);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to take screenshot');
    } finally {
      setIsTakingScreenshot(false);
    }
  };

  const handleTakeProfileScreenshot = async () => {
    setIsTakingProfileScreenshot(true);
    try {
      const response = await api.takeProfileScreenshot();
      if (response.status === 'success') {
        setProfileScreenshotUrl(response.screenshot_url);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to take profile screenshot');
    } finally {
      setIsTakingProfileScreenshot(false);
    }
  };

  const handleSubmitTxHash = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!txHash || !chain) {
      setError('Please enter both transaction hash and chain');
      return;
    }

    setIsSubmitting(true);
    try {
      const response = await api.postTxHash(
        txHash,
        chain,
        selectedWebhookOption !== 'none' ? webhookUrl : undefined
      );
      
      if (response.status === 'success' || response.status === 'processing') {
        // Create a webhook notification from the response
        const webhookNotification: WebhookNotification = {
          status: response.status,
          txHash: response.txHash,
          chain: response.chain,
          fromAddress: response.fromAddress || '',
          fromToken: response.fromToken || '',
          amount: response.amount || '',
          timestamp: Math.floor(Date.now() / 1000).toString(),
          data: response.data || {},
          client: 'twitter-solver'
        };
        
        setWebhookStatus(webhookNotification);
        setTxHash('');
        setChain('');
        setError('');
      } else {
        setWebhookStatus({
          status: 'error',
          txHash: txHash,
          chain: chain,
          fromAddress: '',
          fromToken: '',
          amount: '',
          timestamp: Math.floor(Date.now() / 1000).toString(),
          data: { error: response.message || 'Failed to submit transaction hash' },
          client: 'crowd-funding'
        });
        setError(response.message || 'Failed to submit transaction hash');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'An unexpected error occurred';
      setWebhookStatus({
        status: 'error',
        txHash: txHash,
        chain: chain,
        fromAddress: '',
        fromToken: '',
        amount: '',
        timestamp: Math.floor(Date.now() / 1000).toString(),
        data: { error: errorMessage },
        client: 'twitter-solver'
      });
      if (err instanceof Error) {
        if (err.message.includes('webhook')) {
          setError('Webhook error: ' + err.message);
        } else if (err.message.includes('404')) {
          setError('Transaction hash not found');
        } else {
          setError(err.message);
        }
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleWebhookOptionChange = (optionId: string) => {
    setSelectedWebhookOption(optionId);
    const selectedOption = webhookOptions.find(option => option.id === optionId);
    
    if (selectedOption) {
      if (optionId === 'custom') {
        setUseWebhook(true);
        setWebhookUrl('');
      } else {
        setUseWebhook(optionId !== 'none');
        setWebhookUrl(selectedOption.url);
      }
    }
  };

  if (error) {
    return (
        <Box sx={{ p: 3, textAlign: 'center' }}>
          <Typography color="error" gutterBottom>
            Error: {error}
          </Typography>
          <Button variant="contained" onClick={() => navigate('/')}>
            Return to Home
          </Button>
        </Box>
    );
  }

  return (
    <Box display="flex" flexDirection="column" alignItems="center" minHeight="100vh" p={3}>
      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%', mb: 3 }}>
        <UserProfile />
      </Paper>

      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%', mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Submit Transaction Hash
        </Typography>
        <Box sx={{ mb: 2 }}>
          <TextField
            fullWidth
            label="Transaction Hash"
            value={txHash}
            onChange={(e) => setTxHash(e.target.value)}
            margin="normal"
          />
          <TextField
            fullWidth
            label="Chain"
            value={chain}
            onChange={(e) => setChain(e.target.value)}
            margin="normal"
          />
          <Typography variant="subtitle1" sx={{ mt: 2, mb: 1 }}>
            Webhook Options
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Note: Make sure the webhook URL is accessible before submitting. For testing, you can use the built-in webhook receiver below.
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {webhookOptions.map((option) => (
              <FormControlLabel
                key={option.id}
                control={
                  <Switch
                    checked={selectedWebhookOption === option.id}
                    onChange={() => handleWebhookOptionChange(option.id)}
                  />
                }
                label={option.label}
              />
            ))}
          </Box>
          {selectedWebhookOption === 'custom' && (
            <TextField
              fullWidth
              label="Custom Webhook URL"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              margin="normal"
            />
          )}
          <Button
            variant="contained"
            onClick={handleSubmitTxHash}
            disabled={isSubmitting}
            fullWidth
            sx={{ mt: 2 }}
          >
            {isSubmitting ? 'Submitting...' : 'Submit Transaction Hash'}
          </Button>
        </Box>
        {webhookStatus && (
          <Box sx={{ mt: 2, p: 2, bgcolor: 
            webhookStatus.status === 'error' ? '#FEE2E2' : 
            webhookStatus.status === 'processing' ? '#FEF3C7' : 
            '#F0FDF4', 
            borderRadius: 1 
          }}>
            <Typography variant="subtitle1" gutterBottom>
              Webhook Status:
            </Typography>
            <Typography variant="body2" sx={{ 
              color: 
                webhookStatus.status === 'error' ? '#DC2626' : 
                webhookStatus.status === 'processing' ? '#D97706' : 
                '#059669', 
              fontWeight: 'bold' 
            }}>
              Status: {webhookStatus.status}
            </Typography>
            {webhookStatus.txHash && (
              <Typography variant="body2">
                Transaction Hash: {webhookStatus.txHash}
              </Typography>
            )}
            {webhookStatus.chain && (
              <Typography variant="body2">
                Chain: {webhookStatus.chain}
              </Typography>
            )}
            {webhookStatus.fromAddress && (
              <Typography variant="body2">
                From: {webhookStatus.fromAddress}
              </Typography>
            )}
            {webhookStatus.amount && (
              <Typography variant="body2">
                Amount: {typeof webhookStatus.amount === 'string' ? webhookStatus.amount : JSON.stringify(webhookStatus.amount)} {webhookStatus.fromToken}
              </Typography>
            )}
            {webhookStatus.timestamp && (
              <Typography variant="body2">
                Time: {new Date(parseInt(typeof webhookStatus.timestamp === 'string' ? webhookStatus.timestamp : String(webhookStatus.timestamp)) * 1000).toLocaleString()}
              </Typography>
            )}
            {webhookStatus.client && (
              <Typography variant="body2">
                Client: {webhookStatus.client}
              </Typography>
            )}
            {webhookStatus.data?.error && (
              <Box sx={{ mt: 2, p: 2, bgcolor: '#FEF2F2', borderRadius: 1 }}>
                <Typography variant="body2" sx={{ color: '#DC2626', fontWeight: 'bold' }}>
                  Error Details:
                </Typography>
                <Typography variant="body2" sx={{ color: '#DC2626' }}>
                  {webhookStatus.data.error}
                </Typography>
              </Box>
            )}
            {webhookStatus.data?.screenshot && (
              <Box sx={{ mt: 2 }}>
                <Typography variant="body2" gutterBottom>
                  Screenshot:
                </Typography>
                <img
                  src={`http://localhost:3005/screenshots/${webhookStatus.data.screenshot}`}
                  alt="Transaction Screenshot"
                  style={{ maxWidth: '100%', borderRadius: 4 }}
                />
              </Box>
            )}
          </Box>
        )}
      </Paper>

      {/* Webhook Receiver Component */}
      <WebhookReceiver refreshInterval={2000} />

      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%' }}>
        <Box display="flex" alignItems="center" mb={3}>
          <Avatar sx={{ bgcolor: '#1DA1F2', mr: 2 }}>
            {user?.username[0].toUpperCase()}
          </Avatar>
          <Box>
            <Typography variant="h6">{user?.name}</Typography>
            <Typography color="text.secondary">@{user?.username}</Typography>
          </Box>
        </Box>

        <Box sx={{ mb: 3 }}>
          <Button
            variant="outlined"
            startIcon={<CameraAltIcon />}
            onClick={handleTakeProfileScreenshot}
            disabled={isTakingProfileScreenshot}
            fullWidth
          >
            {isTakingProfileScreenshot ? 'Taking Profile Screenshot...' : 'Take Profile Screenshot'}
          </Button>
          {profileScreenshotUrl && (
            <Box sx={{ mt: 2 }}>
              <img
                src={`http://localhost:3005${profileScreenshotUrl}`}
                alt="Profile Screenshot"
                style={{ maxWidth: '100%', height: 'auto' }}
              />
            </Box>
          )}
        </Box>

        {tweet && (
            <Paper sx={{ p: 3, mt: 3 }}>
              <Typography variant="h6" gutterBottom>
                Latest Tweet
              </Typography>
              <Typography variant="body1" paragraph>
                {tweet.text}
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                Posted on {new Date(tweet.created_at).toLocaleString()}
              </Typography>
              <Box sx={{ mt: 2, display: 'flex', gap: 2 }}>
                <Button
                    variant="outlined"
                    startIcon={<CameraAltIcon />}
                    onClick={handleTakeScreenshot}
                    disabled={isTakingScreenshot}
                >
                  {isTakingScreenshot ? 'Taking Screenshot...' : 'Take Screenshot'}
                </Button>
                <Link
                    href={`https://twitter.com/user/status/${tweet.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                >
                  View on Twitter
                </Link>
              </Box>
              {screenshotUrl && (
                  <Box sx={{ mt: 2 }}>
                    <img
                        src={`http://localhost:3005${screenshotUrl}`}
                        alt="Tweet Screenshot"
                        style={{ maxWidth: '100%', height: 'auto' }}
                    />
                  </Box>
              )}
            </Paper>
        )}
      </Paper>
    </Box>
  );
}; 

