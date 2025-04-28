import * as core from '@actions/core';
import * as github from '@actions/github';

async function run() {
    try {
        const labels_to_add = core.getMultilineInput('add');
        const labels_to_remove = core.getMultilineInput('remove');

        const token = process.env.GITHUB_TOKEN;
        if (!token) {
            throw new Error('missing required environment variable: GITHUB_TOKEN');
        }

        const issue_number = github.context.issue.number;
        if (!issue_number) {
            throw new Error('no issue or pull request context found');
        }

        const event_name = github.context.eventName;
        const event_action = github.context.payload.action;
        console.log(`triggered by: ${event_name}.${event_action} for #${issue_number}`);

        const { owner, repo } = github.context.repo;

        const overlap = labels_to_add.filter(label => labels_to_remove.includes(label));
        if (overlap.length > 0) {
            const formatted = overlap.map(label => `- ${label}`).join('\n');
            throw new Error(`detected conflicting labels:\n\n${formatted}`);
        }

        const octokit = github.getOctokit(token);

        const { data: issue } = await octokit.rest.issues.get({
            owner,
            repo,
            issue_number
        });

        const current_labels = issue.labels.map(label => label.name);

        if (labels_to_add.length > 0) {
            const labels = []

            for (const label of labels_to_add) {
                if (!current_labels.includes(label)) {
                    labels.push(label);
                }
            }

            if (labels.length > 0) {
                await octokit.rest.issues.addLabels({
                    owner,
                    repo,
                    issue_number,
                    labels
                });
            } else {
                console.log(`no new labels to add`)
            }
        }

        if (labels_to_remove.length > 0) {
            for (const label of labels_to_remove) {
                if (current_labels.includes(label)) {
                    console.log(`removing label: ${label}`);
                    await octokit.rest.issues.removeLabel({
                        owner,
                        repo,
                        issue_number,
                        name: label
                    });
                }
            }
        }
    } catch (error) {
        core.setFailed(error.message);
    }
}

run();
