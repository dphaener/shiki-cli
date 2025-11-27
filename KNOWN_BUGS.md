# Known bugs that need to be fixed

## Query error stops agent execution

```
failed to send query: failed to write to stdin: write |1: file already closed
```

## Plan mode doesn't preview template

When in plan mode, the preview pane doesn't show the template after it has been generated

## Tasks mode doesn't preview template

Same as above

## Implement mode should show task list

During implementation we should immediately show the task list and update it in real time as the tasks are implemented.

## AI thinking status still sometimes broken

When the agent takes multiple turns the thinking status disappears after the first turn

## Todo Tool enhancements

The Todo Tool output needs to be more succint and have slightly more information about the tool use. Don't just show a
list of tools used during a turn, replace the tool usage. Much like claude code does.

## Error messaging

The error messaging is still WAY too verbose sometimes. Clean it up. Gather some examples.

```
 Tool failed: <tool_use_error>File has not been read yet. Read it first before                  ││                                                                 │
│   writing to it.</tool_use_error> Tool: Edit Args:                                               ││                                                                 │
│   {"file_path":"/Users/darinhaener/code/collab/internal/tui/plan_model.go","old_string":"con     ││                                                                 │
│   (\n\tPlanChatPane PlanPaneType = iota\n\tPlanPreviewPane\n)","new_string":"const               ││                                                                 │
│   (\n\tPlanChatPane PlanPaneType = iota\n\tPlanPreviewPane\n)\n\n// Keyboard                     ││                                                                 │
│   interaction constants\nconst (\n\t// doubleEscapeTimeout is the time window for                ││                                                                 │
│   detecting double ESC key presses\n\tplanDoubleEscapeTimeout = 2 * time.Second\n\t//            ││                                                                 │
│   messageClearShortcut defines the keyboard combination for clearing                             ││                                                                 │
│   messages\n\tplanMessageClearShortcut = \"ctrl+u\"\n)"}
```

## Create a new bug command

We need a new bug command that is similar to feature but with a less intensive planning session.
