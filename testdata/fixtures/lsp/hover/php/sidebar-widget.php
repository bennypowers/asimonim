<?php
/**
 * Sidebar widget template demonstrating design token hover.
 */
?>
<aside class="widget">
  <style>
    .widget {
      padding: var(--spacing-md);
      border: 1px solid var(--color-border);
    }
    .widget-title {
      font-size: var(--font-size-lg);
      color: var(--color-heading);
    }
  </style>
  <h3 class="widget-title"><?php echo $title; ?></h3>
  <div style="margin-top: var(--spacing-sm)">
    <?php echo $content; ?>
  </div>
</aside>
