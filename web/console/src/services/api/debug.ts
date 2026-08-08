/**
 * API Debug Utilities
 * Helper functions for debugging API connectivity issues
 */

import { API_CONFIG } from './config'

export const debugApiConnection = async () => {
  console.group('🔍 API Connection Debug')
  
  // Log current configuration
  console.log('📋 Current API Configuration:')
  console.log('  Base URL:', API_CONFIG.BASE_URL)
  console.log('  Timeout:', API_CONFIG.TIMEOUT)
  console.log('  Headers:', API_CONFIG.HEADERS)
  console.log('  Environment:', import.meta.env.MODE)
  console.log('  Is Development:', import.meta.env.DEV)
  
  // Test basic connectivity
  console.log('\n🌐 Testing API Connectivity...')
  
  try {
    const testUrl = `${API_CONFIG.BASE_URL}/health`
    console.log('  Testing URL:', testUrl)
    
    const response = await fetch(testUrl, {
      method: 'GET',
      headers: API_CONFIG.HEADERS,
    })
    
    console.log('  Response Status:', response.status)
    console.log('  Response Headers:', Object.fromEntries(response.headers.entries()))
    
    if (response.ok) {
      const data = await response.text()
      console.log('  Response Data:', data)
      console.log('✅ API Connection: SUCCESS')
    } else {
      console.log('❌ API Connection: FAILED')
      console.log('  Status Text:', response.statusText)
    }
  } catch (error) {
    console.log('❌ API Connection: ERROR')
    console.error('  Error Details:', error)
    
    if (error instanceof Error) {
      console.log('  Error Name:', error.name)
      console.log('  Error Message:', error.message)
    }
  }
  
  console.groupEnd()
}

// Auto-run debug in development mode
if (import.meta.env.DEV) {
  console.log('🚀 Development mode detected - API debug info available')
  console.log('💡 Run debugApiConnection() in console to test API connectivity')
  
  // Make it available globally for console debugging
  ;(window as Window & { debugApiConnection?: typeof debugApiConnection }).debugApiConnection =
    debugApiConnection
}
