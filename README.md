# Interruption Tracker

A terminal-based application for tracking work sessions and interruptions, helping you understand and optimize your productivity patterns.

## Features

### Core Features
- Track work sessions with start and end times
- Record interruptions with categorization (calls, meetings, spouse, other)
- View detailed statistics about work patterns and interruptions
- Interactive terminal UI with keyboard shortcuts
- Automatic calculation of work and interruption durations
- Support for session descriptions and interruption notes
- Session resuming and editing capabilities

### Interface & Views
#### Main Session View
- Real-time work session tracking
- Active session status indicators
- Interruption recording interface
- Sortable session history table
- Session details modal with sub-session breakdown
- Interruption categorization dialog

#### Statistics View
- Comprehensive statistics dashboard
- Daily timeline visualization of work patterns
- Completed tasks breakdown
- Interruption analysis by category
- Work efficiency calculation and display
- Recovery time impact analysis

#### Visualization Pages
- **Productivity Visualizations**: Score-based charts and metrics
- **Interruption Analysis**: Detailed breakdown of interruption patterns
- **Productivity Trends**: Time-based analysis showing productivity over days/weeks

### Enhanced Visualization
- Productivity score calculation and analysis (0-100 scale)
- Interactive productivity charts and visualizations
- Hourly productivity heatmap views
- Trend analysis showing productivity patterns over time
- Color-coded statistics with gradient visualization
- Real-time session duration updating

### Data Management
- Configurable data storage location
- Automated data backups
- Data import/export functionality
- Secure session deletion
- Session merging capability
- Command-line utility operations
- Cross-midnight session handling

### Statistics & Analysis
- Daily, weekly, monthly, quarterly, and yearly statistics
- Productivity scoring algorithm with efficiency metrics
- Recovery time impact analysis (10-minute recovery period)
- Hour-by-hour productivity tracking
- Personalized productivity recommendations
- Interruption pattern detection and categorization
- Work efficiency calculations

### Security Features
- Optional data encryption
- Password protection capability
- Secure session deletion

## Installation

```bash
go install github.com/lukaszraczylo/interruption-tracker
```

## Usage

### Basic Usage
```bash
interruption-tracker
```

### Command-line Options
```bash
interruption-tracker --help              # Show all options
interruption-tracker --stats=week        # Display weekly statistics
interruption-tracker --export=data.json  # Export all data to file
interruption-tracker --import=data.json  # Import data from file
interruption-tracker --backup=backup.zip # Create a backup archive
interruption-tracker --version           # Show version information
```

### Keyboard Controls
#### Main View Controls

| Key | Action |
| --- | ------ |
| `s` | Start a new work session |
| `e` | End current session |
| `i` | Record an interruption |
| `b` | Return from interruption |
| `r` | Rename/edit description |
| `d` | Delete selected session |
| `u` | Undo session end (resume) |
| `v` | View statistics |
| `Enter` | Show detailed session information |
| `q` | Quit application |
| `Ctrl+C` | Force quit application |

#### Statistics View Controls

| Key | Action |
| --- | ------ |
| `d` | View daily statistics |
| `w` | View weekly statistics |
| `m` | View monthly statistics |
| `q` | View quarterly statistics |
| `y` | View yearly statistics |
| `a` | View all-time statistics |
| `b` | Return to main view |
| `p` | Show productivity visualizations |
| `t` | Show productivity trends |
| `i` | Show interruption analysis |
| `h` | Alternative for productivity visualizations |
| `v` | Return to main view (alternative) |
| `q` | Quit application |

#### Visualization Controls

| Key | Action |
| --- | ------ |
| `d` | Switch to day view |
| `w` | Switch to week view |
| `m` | Switch to month view |
| `←` | Navigate to previous visualization page |
| `→` | Navigate to next visualization page |
| `b` | Return to main statistics view |
| `q` | Quit application |

#### Modal Dialog Controls

| Key | Action |
| --- | ------ |
| `Enter` | Submit/confirm input |
| `Esc` | Cancel/close dialog |
| `1-4` | Quick selection in interruption type dialog |
| `q` | Quit application |

## Application Views

The Interruption Tracker provides several specialized views to help you track your work and analyze your productivity:

### Main Session View
- **Session Table**: Central display showing all current sessions with start times, end times, durations, and interruption counts
- **Status Bar**: Displays available commands and current application state
- **Active Session Indicator**: Highlights the currently active session
- **Description Input**: Modal for entering or editing session descriptions
- **Sub-sessions**: Tracks continuous work periods within a single logical session
- **Session Details**: Detailed modal view showing session breakdown with sub-sessions and all interruptions

### Statistics View
- **Summary Statistics**: Shows total work time, interruption time, interruption count, and work efficiency
- **Daily Timeline**: Visual 24-hour timeline showing work periods, interruptions, and recovery periods
- **Completed Tasks Table**: Displays finished work sessions with descriptions, durations, and interruption counts
- **Interruption Analysis Table**: Breaks down interruptions by type with counts and durations

### Productivity Visualizations
- **Productivity Score Chart**: Visual representation of work efficiency on a 0-100 scale
- **Hourly Productivity Chart**: Shows productivity patterns throughout the day
- **Day/Week/Month Views**: Ability to view productivity metrics at different time scales
- **Color-coded Timeline**: Instantly identify working periods, interruptions, and recovery times

### Interruption Analysis View
- **Interruption Breakdown Charts**: Visual representation of interruption patterns
- **Category Distribution**: Shows the distribution of different interruption types
- **Impact Analysis**: Visualizes the impact of interruptions on productivity
- **Recovery Time Analysis**: Insights into context-switching costs

### Productivity Trends View
- **Daily Productivity Chart**: Shows productivity scores over multiple days
- **Trend Analysis**: Visual patterns identifying your most and least productive periods
- **Historical Comparison**: Compare current productivity with past periods
- **Multi-day Visualization**: See productivity patterns across longer timeframes

### Interruption Types

The application supports both standard and custom interruption categories:

#### Default Categories
1. Calls
2. Meetings
3. Spouse/Family
4. Other (custom with description)

#### Custom Categories
Custom interruption categories can be defined in the configuration file.

### Statistics Tracking

The application provides comprehensive statistics and metrics:

#### Work Metrics
- **Total Work Duration**: Pure working time excluding interruptions
- **Work Efficiency**: Percentage of productive time relative to total session time
- **Session Analysis**: Breakdown of individual work sessions with durations
- **Sub-session Tracking**: Detailed metrics on continuous work periods within sessions
- **Cross-midnight Handling**: Proper accounting for sessions that span multiple days

#### Interruption Metrics
- **Interruption Count**: Total number of interruptions and breakdown by type
- **Interruption Duration**: Time spent dealing with interruptions
- **Recovery Time**: 10-minute recovery period added after each interruption (configurable)
- **Interruption Tags**: Categorization of interruptions (calls, meetings, spouse, other, custom)
- **Average Duration**: Mean time of interruptions by category

#### Visualization Metrics
- **Daily Timeline**: 24-hour visual timeline showing work and interruption patterns
- **Productivity Score**: Efficiency rating on a 0-100 scale
- **Heatmap Analysis**: Color-coded visualization of most productive hours
- **Productivity Trends**: Patterns showing productivity variations over time
- **Interruption Impact**: Visual representation of how interruptions affect work

#### Time Range Analysis
- **Daily Statistics**: Focused view of today's productivity
- **Weekly Statistics**: Aggregated data for the current week
- **Monthly Statistics**: Broader view of monthly patterns
- **Quarterly Statistics**: Long-term productivity analysis
- **Yearly Statistics**: Annual productivity overview
- **All-time Statistics**: Complete historical data analysis

## Configuration

You can customize the application behavior through a configuration file. The application supports both JSON and YAML formats for configuration.

### Configuration File Locations

The application looks for configuration files in the following locations (in order of priority):

- Linux/macOS: 
  - `~/.interruption-tracker/config.yaml`
  - `~/.interruption-tracker/config.yml`
  - `~/.interruption-tracker/config.json`
  - `~/.config/interruption-tracker/config.yaml`
  - `~/.config/interruption-tracker/config.yml`
  - `~/.config/interruption-tracker/config.json`
- Windows: 
  - `%USERPROFILE%\.interruption-tracker\config.yaml`
  - `%USERPROFILE%\.interruption-tracker\config.yml`
  - `%USERPROFILE%\.interruption-tracker\config.json`
  - `%APPDATA%\interruption-tracker\config.yaml`
  - `%APPDATA%\interruption-tracker\config.yml`
  - `%APPDATA%\interruption-tracker\config.json`

You can also specify a custom configuration file path using the `--config` flag:

```bash
interruption-tracker --config=/path/to/your/config.yaml
```

### Configuration Formats

#### JSON Example
```json
{
  "data_directory": "/custom/path/to/data",
  "backup_enabled": true,
  "backup_interval": 7,
  "recovery_time": 10,
  "enable_mouse": true,
  "color_theme": "dark",
  "custom_interruption_tags": ["Slack", "Email", "Coffee"]
}
```

#### YAML Example
```yaml
data_directory: /custom/path/to/data
backup_enabled: true
backup_interval: 7
recovery_time: 10
enable_mouse: true
color_theme: dark
custom_interruption_tags:
  - Slack
  - Email
  - Coffee
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
