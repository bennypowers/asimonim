<?php
$brand_color = get_theme_mod('brand_color', '#0073aa');
$font_stack = get_theme_mod('font_family', 'system-ui');
?>
<style>
:root {
  --brand-color: <?php echo esc_attr($brand_color); ?>;
  --font-stack: <?php echo esc_attr($font_stack); ?>;
}
.site-header {
  background: var(--brand-color);
  font-family: var(--font-stack);
  color: var(--color-text);
}
<?php if ($has_gradient): ?>
.hero {
  background: linear-gradient(var(--gradient-start), var(--gradient-end));
}
<?php endif; ?>
</style>
<div style="color: var(--color-primary); padding: <?php echo esc_attr($pad); ?>px">
  <?php the_content(); ?>
</div>
<style>
.dynamic-section {
  opacity: <?= $opacity ?>;
  margin: var(--spacing-section);
}
</style>
<?php if ($show_sidebar): ?>
<aside style="width: var(--sidebar-width)">
  <?= $sidebar_content ?>
</aside>
<?php endif; ?>
